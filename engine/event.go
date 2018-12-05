package engine

import (
	"fmt"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func (es *EngineSession) processEvents() (nextEventValid bool, nextEvent time.Time) {
	now := es.timestamp

	err, opt := es.GetOptions()
	if err != nil {
		panic(err)
	}

	lastturn := opt.LastTurnCalculated
	endturn := lastturn.Add(time.Duration(opt.TurnDuration) * time.Minute)

	es.logger.Println("first endturn is ", endturn)

	for lastturn.Before(now) && opt.Turn <= opt.EndGame {
		if now.Before(endturn) {
			endturn = now
		}

		es.logger.Println("doing everything between ", lastturn, " and ", endturn)

		shareauctions := make([]*ShareAuction, 0)
		if err := es.tx.Preload("Share").Preload("HighestOfferPlayer").Where("`expiration` < ?", endturn).Find(&shareauctions).Error; err != nil {
			panic(err)
		}

		for _, sa := range shareauctions {
			sa.Share.OwnerID = sa.HighestOfferPlayerID

			cmp := &Company{}
			if err := es.tx.Where(sa.Share.CompanyID).First(cmp).Error; err != nil {
				panic(err)
			}

			cmp.ShareCapital += sa.HighestOffer

			if err := es.tx.Save(&sa.Share).Error; err != nil {
				panic(err)
			}

			if err := es.tx.Save(cmp).Error; err != nil {
				panic(err)
			}

			participations := make([]*ShareAuctionParticipation, 0)
			es.tx.Model(&ShareAuctionParticipation{}).Where("`share_auction_id` = ?", sa.ID).Find(&participations)

			// report generation

			for _, participant := range participations {
				subject := "Asta " + cmp.Name
				content := fmt.Sprintf("L'asta a cui hai partecipato per la societa' "+cmp.Name+" e' stata vinta da "+sa.HighestOfferPlayer.Name+" per %d$.", sa.HighestOffer)
				report := &Report{PlayerID: participant.PlayerID, Date: sa.Expiration, Subject: subject, Content: content}
				if err := es.tx.Create(report).Error; err != nil {
					panic(err)
				}
			}

			if err := es.tx.Delete(sa).Error; err != nil {
				panic(err)
			}
			if err := es.tx.Delete(&ShareAuctionParticipation{}, "share_auction_id = ?", sa.ID).Error; err != nil {
				panic(err)
			}
		}

		transferproposals := make([]*TransferProposal, 0)
		if err := es.tx.Preload("From").Preload("To").Where("`expiration` < ?", endturn).Find(&transferproposals).Error; err != nil {
			panic(err)
		}

		for _, tp := range transferproposals {
			tp.From.Budget += tp.Amount

			if err := es.tx.Save(&tp.From).Error; err != nil {
				panic(err)
			}

			// report generation

			subject := "Proposta di trasferimento denaro"
			content := fmt.Sprintf("La proposta di trasferimento di %d$ da "+tp.From.Name+" a "+tp.To.Name+" e' scaduta", tp.Amount)
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
		if err := es.tx.Preload("From").Preload("To").Where("`proposal_expiration` < ?", endturn).Find(&partnerships).Error; err != nil {
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
			es.logger.Println("end turn on ", endturn)

			// UPDATE POWER SUPPLIES
			nodesByCoord := make(map[Coord]*Node)
			nodesUpdated := make([]*Node, 0)
			nodes := make([]*Node, 0)

			if err := es.tx.Model(&Node{}).Update("power_supply", PowerOK).Error; err != nil {
				panic(err)
			}

			es.tx.Find(&nodes)

			for _, n := range nodes {
				nodesByCoord[Coord{X: n.X, Y: n.Y}] = n

				if randEvent(0.001 * float64(n.Yield)) {
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
				err, pureIncome, valuePerShare := es.GetCompanyFinancials(cmp)
				if err != nil {
					panic(err)
				}

				shareholders := make([]*ShareholdersPerCompany, 0)

				shares := 0

				if err := es.tx.Table("shares").Select("DISTINCT owner_id, count(owner_id) as shares").Where("`company_id` = ?", cmp.ID).Where("`owner_id` != 0").Group("owner_id").Find(&shareholders).Error; err != nil {
					panic(err)
				}

				for _, sh := range shareholders {
					shares += sh.Shares
				}

				for _, sh := range shareholders {
					shareholder.ID = 0

					if err := es.tx.Where(sh.OwnerID).Find(shareholder).Error; err != nil {
						panic(err)
					}

					shareholder.Budget += valuePerShare * sh.Shares
					shareholder.LastIncome += valuePerShare * sh.Shares
					shareholder.LastBudget = shareholder.Budget

					if err := es.tx.Save(shareholder).Error; err != nil {
						panic(err)
					}

					dividendsPerPlayer[sh.OwnerID] = append(dividendsPerPlayer[sh.OwnerID], Dividend{cmp, valuePerShare * sh.Shares})
				}

				cmp.ShareCapital += int(pureIncome)
				cmp.ActionPoints = opt.CompanyActionPoints + shares

				if err := es.tx.Save(cmp).Error; err != nil {
					panic(err)
				}
			}

			for shid, dividends := range dividendsPerPlayer {
				subject := fmt.Sprintf("Dividendi turno %d", opt.Turn)
				content := fmt.Sprintf("I dividendi per questo turno sono i seguenti.<br>")

				totalincome := 0
				for _, d := range dividends {
					content += fmt.Sprintf(d.Company.Name+" : %d$<br>", d.Income)
					totalincome += d.Income
				}

				content += fmt.Sprintf("Totale: %d$", totalincome)

				report := &Report{PlayerID: shid, Date: endturn, Subject: subject, Content: content}

				if err := es.tx.Create(report).Error; err != nil {
					panic(err)
				}
			}

			if err := es.tx.Model(&Player{}).Update("action_points", opt.PlayerActionPoints).Error; err != nil {
				panic(err)
			}

			opt.LastTurnCalculated = endturn
			opt.Turn += 1
		}

		lastturn = endturn
		endturn = endturn.Add(time.Duration(opt.TurnDuration) * time.Minute)
	}

	if err := es.tx.Save(&opt).Error; err != nil {
		panic(err)
	}

	// calculate next event timestamp; by default it is the turn end...
	nextEventValid, nextEvent = true, opt.LastTurnCalculated.Add(time.Duration(opt.TurnDuration)*time.Minute)

	// ... then we check auctions...
	shareauction := ShareAuction{}

	err = es.tx.Order("`expiration`").First(&shareauction).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if !shareauction.Expiration.IsZero() && shareauction.Expiration.Before(nextEvent) {
		nextEvent = shareauction.Expiration
	}

	// ... then we check transfer proposals...
	transferproposal := TransferProposal{}

	err = es.tx.Order("`expiration`").First(&transferproposal).Error

	if err != nil && err != gorm.ErrRecordNotFound {
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
	if opt.Turn > opt.EndGame {
		nextEventValid = false
	}

	return
}

func randEvent(p float64) bool {
	randint := rand.Intn(1000) + 1
	randf := float64(randint) / 1000.0

	return randf < p
}
