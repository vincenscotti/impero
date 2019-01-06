package engine

import (
	"errors"
	. "github.com/vincenscotti/impero/model"
)

func (es *EngineSession) GetChatMessages(p *Player) (err error, messages []*ChatMessage) {
	messages = make([]*ChatMessage, 0)

	if err := es.tx.Preload("From").Order("Date asc", true).Find(&messages).Error; err != nil {
		panic(err)
	}

	p.LastChatViewed = es.timestamp
	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	return
}

func (es *EngineSession) PostChatMessage(p *Player, text string) error {
	msg := &ChatMessage{}

	msg.Content = text

	if msg.Content == "" {
		return errors.New("Messaggio vuoto non valido!")
	}

	msg.FromID = p.ID
	msg.Date = es.timestamp

	if err := es.tx.Create(msg).Error; err != nil {
		panic(err)
	}

	return nil
}
