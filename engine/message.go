package engine

import (
	. "github.com/vincenscotti/impero/model"
)

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
