package engine

import (
	. "github.com/vincenscotti/impero/model"
)

type Notificator interface {
	NotifyAuctionRaise(company *Company, auction *ShareAuction, players []*Player)
	NotifyAuctionEnd(company *Company, auction *ShareAuction, players []*Player)
	NotifyEndTurn()
}

func (e *Engine) RegisterNotificator(n Notificator) {
	e.notificator = n
}
