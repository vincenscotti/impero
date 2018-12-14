package engine

import (
	"errors"
	. "github.com/vincenscotti/impero/model"
)

func (es *EngineSession) GetInbox(p *Player) (err error, messages []*Message) {
	messages = make([]*Message, 0)
	if err := es.tx.Where("`to_id` = ?", p.ID).Preload("From").Order("`Date` desc", true).Find(&messages).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) GetOutbox(p *Player) (err error, messages []*Message) {
	messages = make([]*Message, 0)
	if err := es.tx.Where("`from_id` = ?", p.ID).Preload("To").Order("`Date` desc", true).Find(&messages).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) GetMessage(p *Player, id int) (err error, message *Message) {
	message = &Message{}
	if err := es.tx.Preload("From").Preload("To").Where(id).First(message).Error; err != nil {
		panic(err)
	}

	if message.FromID != p.ID && message.ToID != p.ID {
		err = errors.New("Non puoi leggere questo messaggio!")

		return
	} else if message.ToID == p.ID {
		message.Read = true
		if err := es.tx.Save(&message).Error; err != nil {
			panic(err)
		}
	}

	return
}

func (es *EngineSession) PostMessage(from *Player, to *Player, subject string, content string) error {
	msg := &Message{ToID: to.ID, Content: content, Subject: subject}

	if msg.Content == "" {
		return errors.New("Non puoi inviare un messaggio vuoto!")
	}

	if msg.ToID == 0 {
		return errors.New("Destinatario non valido!")
	}

	msg.FromID = from.ID
	msg.Date = es.timestamp
	msg.Read = false

	if err := es.tx.Create(msg).Error; err != nil {
		panic(err)
	}

	return nil
}

func (es *EngineSession) BroadcastMessage(m *Message) error {
	players := make([]*Player, 0)

	if err := es.tx.Find(&players).Error; err != nil {
		panic(err)
	}

	for _, p := range players {
		m.ID = 0
		m.ToID = p.ID

		if err := es.tx.Create(m).Error; err != nil {
			panic(err)
		}
	}

	return nil
}
