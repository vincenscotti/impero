package engine

import (
	"errors"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"time"
)

func (es *EngineSession) GetSharesForPlayer(p *Player) (err error, shares []*SharesPerPlayer) {
	shares = make([]*SharesPerPlayer, 0)

	if err := es.tx.Table("shares").Select("DISTINCT company_id, count(company_id) as shares").Where("`owner_id` = ?", p.ID).Group("`company_id`").Order("`shares` desc").Find(&shares).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) CalculateSharesIncome(shares []*SharesPerPlayer) (err error, income int) {
	for _, sh := range shares {
		cmp := &sh.Company

		if err := es.tx.Where(sh.CompanyID).Find(cmp).Error; err != nil {
			panic(err)
		}

		err, _, sh.ValuePerShare = es.GetCompanyFinancials(cmp, false)

		if err != nil {
			return
		}

		income += sh.Shares * sh.ValuePerShare
	}

	return
}

func (es *EngineSession) GetShareAuctionsWithPlayerParticipation(p *Player) (err error, shareauctions []*ShareAuction) {
	shareauctions = make([]*ShareAuction, 0)

	if err := es.tx.Model(&ShareAuction{}).Preload("Company").Order("`expiration`").Find(&shareauctions).Error; err != nil {
		panic(err)
	}

	for _, sa := range shareauctions {
		participations := make([]*ShareAuctionParticipation, 0)
		if err := es.tx.Where("`share_auction_id` = ? and `player_id` = ?", sa.ID, p.ID).Find(&participations).Error; err != nil {
			panic(err)
		}

		sa.Participations = participations
	}

	return
}

func (es *EngineSession) CreateAuction(p *Player, cmp *Company, numshares int, price int) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if err := es.tx.First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		return errors.New("Societa' inesistente!")
	}

	if numshares < 1 || numshares > 10 {
		return errors.New("Devi emettere un numero di azioni tra 1 e 10!")
	}

	if cmp.CEOID != p.ID {
		return errors.New("Permessi insufficienti!")
	}

	if cmp.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	cmp.ActionPoints -= 1
	if err := es.tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	for ; numshares > 0; numshares-- {
		if err := es.tx.Create(&ShareAuction{CompanyID: cmp.ID, HighestOffer: price, Expiration: es.timestamp.Add(time.Duration(es.opt.TurnDuration) * time.Minute)}).Error; err != nil {
			panic(err)
		}
	}

	return nil
}

func (es *EngineSession) BidAuction(p *Player, shareauction *ShareAuction, amount int) error {
	oldp := &Player{}
	participation := &ShareAuctionParticipation{}
	participation.ShareAuctionID = shareauction.ID
	participation.PlayerID = p.ID

	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if err := es.tx.Where(participation).Find(participation).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if err := es.tx.First(shareauction).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if shareauction.ID == 0 {
		return errors.New("L'asta non esiste!")
	}

	if shareauction.HighestOffer >= amount {
		return errors.New("Puntata troppo bassa!")
	}

	if (shareauction.HighestOfferPlayerID != p.ID && amount > p.Budget) ||
		(shareauction.HighestOfferPlayerID == p.ID &&
			amount > p.Budget+shareauction.HighestOffer) {
		return errors.New("Budget insufficiente!")
	}

	if participation.ID == 0 {
		if p.ActionPoints < 1 {
			return errors.New("Punti operazione insufficienti!")
		}

		if err := es.tx.Save(participation).Error; err != nil {
			panic(err)
		}

		p.ActionPoints -= 1
	}

	p.Budget -= amount
	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Where(shareauction.HighestOfferPlayerID).Find(oldp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if oldp.ID != 0 {
		oldp.Budget += shareauction.HighestOffer
		if err := es.tx.Save(oldp).Error; err != nil {
			panic(err)
		}
	}

	shareauction.HighestOffer = amount
	shareauction.HighestOfferPlayerID = p.ID

	if shareauction.Expiration.Sub(es.timestamp).Minutes() < 1. {
		shareauction.Expiration = es.timestamp.Add(time.Minute)
	}

	if err := es.tx.Save(shareauction).Error; err != nil {
		panic(err)
	}

	return nil
}
