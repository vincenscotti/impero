package engine

import (
	"errors"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"math"
	"math/rand"
	"time"
)

func (es *EngineSession) GetPlayers() (err error, players []*Player) {
	players = make([]*Player, 0)

	if err := es.tx.Order("last_budget desc").Find(&players).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) GetPlayer(id int) (err error, p *Player) {
	p = &Player{}
	if err := es.tx.Where(id).First(p).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if p.ID == 0 {
		err = errors.New("Giocatore inesistente!")
	}

	return
}

func (es *EngineSession) GetPlayerNotifications(id int) (err error, newchats, newmsgs, newreports int) {
	var p *Player

	if err, p = es.GetPlayer(id); err != nil {
		panic(err)
	}

	if p.ID == 0 {
		err = errors.New("Giocatore inesistente!")
	}

	if err := es.tx.Model(&ChatMessage{}).Where("`date` > ? and `from_id` != ?",
		p.LastChatViewed, p.ID).Count(&newchats).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Model(&Message{}).Where("`read` = ? and `to_id` = ?", false,
		p.ID).Count(&newmsgs).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Model(&Report{}).Where("`read` = ? and `player_id` = ?", false,
		p.ID).Count(&newreports).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) GetIncomingTransfers(p *Player) (err error, incomingTransfers []*TransferProposal) {
	incomingTransfers = make([]*TransferProposal, 0)

	if err := es.tx.Where("`to_id` = ?", p.ID).Preload("From").Find(&incomingTransfers).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) CreateTransferProposal(from, to *Player, amount int) (err error, proposal *TransferProposal) {
	proposal = &TransferProposal{FromID: from.ID, ToID: to.ID, Amount: amount}
	err, opt := es.GetOptions()

	if err != nil {
		return err, nil
	}

	if es.timestamp.Before(opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!"), nil
	}

	if proposal.Amount > from.Budget {
		return errors.New("Budget insufficiente!"), nil
	}

	if from.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!"), nil
	}

	proposal.Risk = int(math.Floor(float64(proposal.Amount) / float64(from.Budget) * 100))
	proposal.Expiration = es.timestamp.Add(time.Duration(opt.TurnDuration) * time.Minute)

	from.Budget -= proposal.Amount
	from.ActionPoints -= 1
	if err := es.tx.Save(from).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Create(proposal).Error; err != nil {
		panic(err)
	}

	return nil, proposal
}

func (es *EngineSession) ConfirmTransferProposal(proposal *TransferProposal) (err error, fiscalCheck bool) {
	p := &TransferProposal{}

	_, opt := es.GetOptions()

	if es.timestamp.Before(opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!"), false
	}

	randint := rand.Intn(100) + 1

	if err := es.tx.Where(proposal.ID).Preload("From").Preload("To").Find(p).Error; err != nil {
		panic(err)
	}

	if p.To.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!"), false
	}

	p.To.ActionPoints -= 1

	if randint < p.Risk {
		// oops
		p.To.Budget = 0

		p.From.Budget = 0
		if err := es.tx.Save(p.From); err.Error != nil {
			panic(err.Error)
		}

		fiscalCheck = true
	} else {
		// success
		p.To.Budget += p.Amount
	}

	if err := es.tx.Save(p.To).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Delete(p).Error; err != nil {
		panic(err)
	}

	return
}
