package engine

import (
	. "github.com/vincenscotti/impero/model"
)

func (es *EngineSession) GetIncomingTransfers(p *Player) (err error, incomingTransfers []*TransferProposal) {
	incomingTransfers = make([]*TransferProposal, 0)

	if err := es.tx.Where("`to_id` = ?", p.ID).Preload("From").Find(&incomingTransfers).Error; err != nil {
		panic(err)
	}

	return
}
