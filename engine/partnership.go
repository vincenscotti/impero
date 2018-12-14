package engine

import (
	"errors"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"time"
)

func (es *EngineSession) ProposePartnership(ceo *Player, cmp1 *Company, cmp2 *Company) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	p := &Partnership{}
	from := &Company{}
	to := &Company{}
	count := 0

	p.FromID = cmp1.ID
	p.ToID = cmp2.ID

	if err := es.tx.Where(p.FromID).First(from).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if err := es.tx.Where(p.ToID).First(to).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	subject := "Proposta di partnership"
	content := "Hai ricevuto una proposta di partnership tra " + from.Name + " e " + to.Name
	report := &Report{PlayerID: to.CEOID, Date: es.timestamp, Subject: subject, Content: content}

	if from.ID == 0 || to.ID == 0 {
		return errors.New("Societa' inesistente!")
	}

	if from.CEOID != ceo.ID {
		return errors.New("Non sei il CEO!")
	}

	if from.CEOID == to.CEOID {
		return errors.New("Sei il CEO di entrambe le societa'!")
	}

	if from.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	if err := es.tx.Table("partnerships").Where("((`from_id` = ? and `to_id` = ?) or (`from_id` = ? and `to_id` = ?)) and `deleted_at` is null", from.ID, to.ID, to.ID, from.ID).Count(&count).Error; err != nil {
		panic(err)
	}

	if count > 0 {
		return errors.New("Partnership o proposta gia' esistente!")
	}

	from.ActionPoints -= 1
	if err := es.tx.Save(from).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Create(&Partnership{FromID: from.ID, ToID: to.ID,
		ProposalExpiration: es.timestamp.Add(time.Duration(es.opt.TurnDuration) * time.Minute)}).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Create(report).Error; err != nil {
		panic(err)
	}

	return nil
}

func (es *EngineSession) ConfirmPartnership(ceo *Player, p *Partnership) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if err := es.tx.Preload("To").Preload("From").First(p).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if p.To.CEOID != ceo.ID {
		return errors.New("Non sei il CEO!")
	}

	if p.To.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	p.To.ActionPoints -= 1
	if err := es.tx.Save(&p.To).Error; err != nil {
		panic(err)
	}

	p.ProposalAccepted = true

	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	subject := "Proposta di partnership confermata"
	content := "La proposta di partnership tra " + p.From.Name + " e " + p.To.Name + " e' stata confermata"
	report := &Report{PlayerID: p.From.CEOID, Date: es.timestamp, Subject: subject, Content: content}

	if err := es.tx.Create(report).Error; err != nil {
		panic(err)
	}

	report.ID = 0
	report.PlayerID = p.To.CEOID

	if err := es.tx.Create(report).Error; err != nil {
		panic(err)
	}

	return nil
}

func (es *EngineSession) DeletePartnership(ceo *Player, p *Partnership) error {
	var cmp *Company

	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if err := es.tx.Preload("To").Preload("From").First(p).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if p.To.CEOID == ceo.ID {
		cmp = &p.To
	} else if p.From.CEOID == ceo.ID {
		cmp = &p.From
	} else {
		return errors.New("Non sei il CEO!")
	}

	if cmp.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	cmp.ActionPoints -= 1
	if err := es.tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Delete(p).Error; err != nil {
		panic(err)
	}

	subject := "Partnership cancellata"
	content := "La partnership tra " + p.From.Name + " e " + p.To.Name + " e' stata cancellata"
	report := &Report{PlayerID: p.From.CEOID, Date: es.timestamp, Subject: subject, Content: content}

	if err := es.tx.Create(report).Error; err != nil {
		panic(err)
	}

	report.ID = 0
	report.PlayerID = p.To.CEOID

	if err := es.tx.Create(report).Error; err != nil {
		panic(err)
	}

	return nil
}
