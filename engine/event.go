package engine

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
)

func init() {
	// FIXME: the random number generator should be injected when building an Engine.
	rand.Seed(time.Now().Unix())
}

func (es *EngineSession) processEvents() (nextEventValid bool, nextEvent time.Time) {
	now := es.timestamp

	lastturn := es.opt.LastTurnCalculated
	endturn := lastturn.Add(time.Duration(es.opt.TurnDuration) * time.Minute)

	es.e.logger.Println("first endturn is ", endturn)

	for lastturn.Before(now) && es.opt.Turn <= es.opt.EndGame {
		if now.Before(endturn) {
			endturn = now
		}

		es.e.logger.Println("doing everything between ", lastturn, " and ", endturn)

		shareauctions := make([]*ShareAuction, 0)
		if err := es.tx.Preload("Company").Preload("HighestOfferPlayer").Where("`expiration` <= ?", endturn).Find(&shareauctions).Error; err != nil {
			panic(err)
		}

		for _, sa := range shareauctions {
			if sa.HighestOfferPlayerID != 0 {
				sh := &Shareholder{}
				sh.CompanyID = sa.CompanyID
				sh.PlayerID = sa.HighestOfferPlayerID
				if err := es.tx.Where(sh).First(sh).Error; err != nil && err != gorm.ErrRecordNotFound {
					panic(err)
				}

				if sh.ID == 0 {
					sh.Shares = 1
				} else {
					sh.Shares += 1
				}

				if err := es.tx.Save(&sh).Error; err != nil {
					panic(err)
				}

				sa.Company.ShareCapital += sa.HighestOffer
			}

			if err := es.tx.Save(sa.Company).Error; err != nil {
				panic(err)
			}

			participations := make([]*ShareAuctionParticipation, 0)
			es.tx.Model(&ShareAuctionParticipation{}).Where("`share_auction_id` = ?", sa.ID).Find(&participations)

			// report generation

			for _, participant := range participations {
				subject := "Asta " + sa.Company.Name
				content := fmt.Sprintf("L'asta a cui hai partecipato per la societa' "+sa.Company.Name+" e' stata vinta da "+sa.HighestOfferPlayer.Name+" per %d $.", sa.HighestOffer/100)
				report := &Report{PlayerID: participant.PlayerID, Date: sa.Expiration, Subject: subject, Content: content}
				if err := es.tx.Create(report).Error; err != nil {
					panic(err)
				}

				player := &Player{}
				player.ID = participant.PlayerID
				es.e.notificator.NotifyAuctionEnd(sa, []*Player{player})
			}

			if err := es.tx.Delete(sa).Error; err != nil {
				panic(err)
			}
			if err := es.tx.Delete(&ShareAuctionParticipation{}, "share_auction_id = ?", sa.ID).Error; err != nil {
				panic(err)
			}
		}

		// delete expired share offers
		if err := es.tx.Delete(&ShareOffer{}, "`expiration` <= ?", endturn).Error; err != nil {
			panic(err)
		}

		transferproposals := make([]*TransferProposal, 0)
		if err := es.tx.Preload("From").Preload("To").Where("`expiration` <= ?", endturn).Find(&transferproposals).Error; err != nil {
			panic(err)
		}

		for _, tp := range transferproposals {
			tp.From.Budget += tp.Amount

			if err := es.tx.Save(&tp.From).Error; err != nil {
				panic(err)
			}

			// report generation

			subject := "Proposta di trasferimento denaro"
			content := fmt.Sprintf("La proposta di trasferimento di %d $ da "+tp.From.Name+" a "+tp.To.Name+" e' scaduta", tp.Amount/100)
			report := &Report{PlayerID: tp.FromID, Date: tp.Expiration, Subject: subject, Content: content}

			if err := es.tx.Create(report).Error; err != nil {
				panic(err)
			}

			report.ID = 0
			report.PlayerID = tp.ToID

			if err := es.tx.Create(report).Error; err != nil {
				panic(err)
			}

			if err := es.tx.Delete(tp).Error; err != nil {
				panic(err)
			}
		}

		partnerships := make([]*Partnership, 0)
		if err := es.tx.Preload("From").Preload("To").Where("`proposal_expiration` <= ?", endturn).Find(&partnerships).Error; err != nil {
			panic(err)
		}

		for _, p := range partnerships {
			if !p.ProposalAccepted {
				// report generation

				subject := "Proposta di partnership scaduta"
				content := "La proposta di partnership tra " + p.From.Name + " e " + p.To.Name + " e' scaduta"
				report := &Report{PlayerID: p.From.CEOID, Date: p.ProposalExpiration, Subject: subject, Content: content}

				if err := es.tx.Create(report).Error; err != nil {
					panic(err)
				}

				report.ID = 0
				report.PlayerID = p.To.CEOID

				if err := es.tx.Create(report).Error; err != nil {
					panic(err)
				}

				if err := es.tx.Delete(p).Error; err != nil {
					panic(err)
				}
			}
		}

		if endturn.Before(now) {
			es.e.logger.Println("end turn on ", endturn)

			// UPDATE POWER SUPPLIES
			nodesByCoord := make(map[Coord]*Node)
			nodesUpdated := make([]*Node, 0)
			nodes := make([]*Node, 0)

			if err := es.tx.Model(&Node{}).Update("power_supply", PowerOK).Error; err != nil {
				panic(err)
			}

			es.tx.Find(&nodes)

			for _, n := range nodes {
				es.updateBlackoutP(n)
				nodesByCoord[Coord{X: n.X, Y: n.Y}] = n

				if randEvent(n.BlackoutProb) {
					n.PowerSupply = PowerOff
					nodesUpdated = append(nodesUpdated, n)
				}
			}

			for _, n := range nodesUpdated {
				adjacentx := []int{n.X - 1, n.X, n.X + 1}
				adjacenty := []int{n.Y - 1, n.Y, n.Y + 1}

				for _, x := range adjacentx {
					for _, y := range adjacenty {
						if x != n.X || y != n.Y {
							if neighbour, ok := nodesByCoord[Coord{X: x, Y: y}]; ok {
								if neighbour.PowerSupply != PowerOff {
									neighbour.PowerSupply = PowerOffNeighbour

									es.tx.Save(neighbour)
								}
							}
						}
					}
				}

				es.tx.Save(n)
			}

			// NOW COMPANY INCOMES
			cmps := make([]*Company, 0)
			shareholder := &Player{}

			es.tx.Find(&cmps)

			type Dividend struct {
				Company *Company
				Income  int
			}
			dividendsPerPlayer := make(map[uint][]Dividend)

			if err := es.tx.Model(&Player{}).Update("last_income", 0).Error; err != nil {
				panic(err)
			}

			for _, cmp := range cmps {
				err, pureIncome, valuePerShare := es.GetCompanyFinancials(cmp, true)
				if err != nil {
					panic(err)
				}

				shareholders := make([]*Shareholder, 0)

				shares := 0

				if err := es.tx.Model(cmp).Related(&shareholders).Error; err != nil {
					panic(err)
				}

				for _, sh := range shareholders {
					shares += sh.Shares
				}

				for _, sh := range shareholders {
					shareholder.ID = 0

					if err := es.tx.Where(sh.PlayerID).Find(shareholder).Error; err != nil {
						panic(err)
					}

					shareholder.Budget += valuePerShare * sh.Shares
					shareholder.LastIncome += valuePerShare * sh.Shares
					shareholder.LastBudget = shareholder.Budget

					if err := es.tx.Save(shareholder).Error; err != nil {
						panic(err)
					}

					dividendsPerPlayer[sh.PlayerID] = append(dividendsPerPlayer[sh.PlayerID], Dividend{cmp, valuePerShare * sh.Shares})
				}

				cmp.ShareCapital += int(pureIncome)
				cmp.ActionPoints = es.opt.CompanyActionPoints + len(shareholders)

				if err := es.tx.Save(cmp).Error; err != nil {
					panic(err)
				}
			}

			for shid, dividends := range dividendsPerPlayer {
				subject := fmt.Sprintf("Dividendi turno %d", es.opt.Turn)
				content := fmt.Sprintf("I dividendi per questo turno sono i seguenti.<br>")

				totalincome := 0
				for _, d := range dividends {
					content += fmt.Sprintf(d.Company.Name+" : %d.%02d $<br>", d.Income/100, d.Income%100)
					totalincome += d.Income
				}

				content += fmt.Sprintf("<br>Totale: %d.%02d $", totalincome/100, totalincome%100)

				report := &Report{PlayerID: shid, Date: endturn, Subject: subject, Content: content}

				if err := es.tx.Create(report).Error; err != nil {
					panic(err)
				}
			}

			if err := es.tx.Model(&Player{}).Update("action_points", es.opt.PlayerActionPoints).Error; err != nil {
				panic(err)
			}

			es.opt.LastTurnCalculated = endturn
			es.opt.Turn += 1

			es.e.notificator.NotifyEndTurn()

			// check if we just calculated the last turn, and updated VP
			if es.opt.Turn > es.opt.EndGame {
				players := make([]*Player, 0)

				if err := es.tx.Find(&players).Error; err != nil {
					panic(err)
				}

				for _, p := range players {
					p.VP = p.Budget
					es.tx.Save(p)
				}
			}
		}

		lastturn = endturn
		endturn = endturn.Add(time.Duration(es.opt.TurnDuration) * time.Minute)
	}

	if err := es.tx.Save(&es.opt).Error; err != nil {
		panic(err)
	}

	// calculate next event timestamp; by default it is the game start...
	nextEventValid, nextEvent = true, es.opt.GameStart

	// ... then we check the turn end...
	if es.opt.GameStart.Before(now) {
		nextEventValid, nextEvent = true, es.opt.LastTurnCalculated.Add(time.Duration(es.opt.TurnDuration)*time.Minute)
	}

	// ... then we check auctions...
	shareauction := ShareAuction{}

	if err := es.tx.Order("`expiration`").First(&shareauction).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if !shareauction.Expiration.IsZero() && shareauction.Expiration.Before(nextEvent) {
		nextEvent = shareauction.Expiration
	}

	// ... then we check offers...
	shareoffer := ShareOffer{}

	if err := es.tx.Order("`expiration`").First(&shareoffer).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if !shareoffer.Expiration.IsZero() && shareoffer.Expiration.Before(nextEvent) {
		nextEvent = shareoffer.Expiration
	}

	// ... then we check transfer proposals...
	transferproposal := TransferProposal{}

	if err := es.tx.Order("`expiration`").First(&transferproposal).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if !transferproposal.Expiration.IsZero() && transferproposal.Expiration.Before(nextEvent) {
		nextEvent = transferproposal.Expiration
	}

	// ... then we check company partnerships...
	partnership := Partnership{}

	if err := es.tx.Where("`proposal_accepted` = false").Order("`proposal_expiration`").First(&partnership).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if !partnership.ProposalExpiration.IsZero() && partnership.ProposalExpiration.Before(nextEvent) {
		nextEvent = transferproposal.Expiration
	}

	// if the game is over, we invalidate the next event
	if es.opt.Turn > es.opt.EndGame {
		nextEventValid = false
	}

	return
}

func randEvent(p float64) bool {
	randint := rand.Intn(1000) + 1
	randf := float64(randint) / 1000.0

	return randf < p
}
