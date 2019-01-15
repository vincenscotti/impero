package engine

import (
	. "github.com/vincenscotti/impero/model"
)

type Notificator interface {
	NotifyAuctionRaise(auction *ShareAuction, players []*Player)
	NotifyAuctionEnd(auction *ShareAuction, players []*Player)
	NotifyEndTurn()
}

func (e *Engine) RegisterNotificator(n Notificator) {
	e.notificator = n
}
