package engine

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"time"
)

func (es *EngineSession) GetSharesForPlayer(p *Player) (err error, shares []*Shareholder) {
	shares = make([]*Shareholder, 0)

	if err := es.tx.Model(shares).Preload("Player").Preload("Company").
		Where("`player_id` = ?", p.ID).Order("`shares` desc").Find(&shares).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) CalculateSharesIncome(shares []*Shareholder) (err error, shpp []*SharesPerPlayer, income int) {
	shpp = make([]*SharesPerPlayer, 0, len(shares))

	for _, sh := range shares {
		out := &SharesPerPlayer{}

		if err := es.tx.Where(sh.CompanyID).Find(&out.ShareholderInfo.Company).Error; err != nil {
			panic(err)
		}

		err, _, out.ValuePerShare = es.GetCompanyFinancials(&out.ShareholderInfo.Company, false)

		if err != nil {
			return
		}

		out.ShareholderInfo = *sh
		shpp = append(shpp, out)

		income += sh.Shares * out.ValuePerShare
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

func (es *EngineSession) GetShareOffers() (err error, shareoffers []*ShareOffer) {
	shareoffers = make([]*ShareOffer, 0)

	if err := es.tx.Model(&ShareOffer{}).Preload("Company").Order("`expiration`").Find(&shareoffers).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) CreateAuction(p *Player, cmp *Company, numshares int, price int) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
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

func (es *EngineSession) SellShares(p *Player, cmp *Company, numshares int, price int) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
	}

	if err := es.tx.First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		return errors.New("Societa' inesistente!")
	}

	offers := 0
	if err := es.tx.Model(&ShareOffer{}).Where("`company_id` = ? and `owner_id` = ?", cmp.ID, p.ID).Count(&offers).Error; err != nil {
		return err
	}

	sh := &Shareholder{}
	sh.CompanyID = cmp.ID
	sh.PlayerID = p.ID
	if err := es.tx.Where(sh).Find(sh).Error; err != nil {
		return err
	}

	shares := sh.Shares

	if offers+numshares > shares {
		return errors.New("Non hai cosi' tante azioni da vendere!")
	}

	if numshares < 1 || numshares > 10 {
		return errors.New("Puoi vendere un numero di azioni tra 1 e 10!")
	}

	if p.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	p.ActionPoints -= 1
	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	for ; numshares > 0; numshares-- {
		if err := es.tx.Create(&ShareOffer{CompanyID: cmp.ID, OwnerID: p.ID, Price: price, Expiration: es.timestamp.Add(time.Duration(es.opt.TurnDuration) * time.Minute)}).Error; err != nil {
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

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
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

	es.e.notificator.NotifyAuctionRaise(nil, shareauction, nil)

	return nil
}

func (es *EngineSession) BuyShare(p *Player, shareoffer *ShareOffer) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
	}

	if err := es.tx.Preload("Company").First(shareoffer).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if shareoffer.ID == 0 {
		return errors.New("L'offerta non esiste!")
	}

	if p.Budget < shareoffer.Price {
		return errors.New("Budget insufficiente!")
	}

	if p.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	p.ActionPoints -= 1

	p.Budget -= shareoffer.Price
	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	oldp := &Player{}
	oldp.ID = shareoffer.OwnerID
	if err := es.tx.First(oldp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if oldp.ID == 0 {
		return errors.New("Proprietario dell'azione non trovato!")
	}

	oldp.Budget += shareoffer.Price
	if err := es.tx.Save(oldp).Error; err != nil {
		panic(err)
	}

	subject := "Vendita azione"
	content := fmt.Sprintf("L'azione della societa' "+shareoffer.Company.Name+" e' stata venduta per %d $.", shareoffer.Price/100)
	report := &Report{PlayerID: oldp.ID, Date: es.timestamp, Subject: subject, Content: content}
	if err := es.tx.Create(report).Error; err != nil {
		panic(err)
	}

	oldsh := &Shareholder{}
	oldsh.CompanyID = shareoffer.CompanyID
	oldsh.PlayerID = shareoffer.OwnerID
	if err := es.tx.Where(oldsh).First(oldsh).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	oldsh.Shares -= 1

	if oldsh.Shares == 0 {
		if err := es.tx.Delete(oldsh).Error; err != nil {
			panic(err)
		}
	} else {
		if err := es.tx.Save(oldsh).Error; err != nil {
			panic(err)
		}
	}

	newsh := &Shareholder{}
	newsh.CompanyID = shareoffer.CompanyID
	newsh.PlayerID = p.ID

	if err := es.tx.Where(newsh).First(newsh).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if newsh.ID == 0 {
		newsh.Shares = 1
	} else {
		newsh.Shares += 1
	}

	if err := es.tx.Save(newsh).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Delete(shareoffer).Error; err != nil {
		panic(err)
	}

	return nil
}
