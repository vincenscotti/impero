package tgui

import (
	. "github.com/vincenscotti/impero/model"
)

type gameNotification interface {
}

type endTurnNotification struct {
}

type auctionRaiseNotification struct {
	auction *ShareAuction
	players []*Player
}

type auctionEndNotification struct {
	auction *ShareAuction
	players []*Player
}

func (tg *TGUI) NotifyAuctionRaise(auction *ShareAuction, players []*Player) {
	tg.notifications <- auctionRaiseNotification{auction: auction, players: players}
}

func (tg *TGUI) NotifyAuctionEnd(auction *ShareAuction, players []*Player) {
	tg.notifications <- auctionEndNotification{auction: auction, players: players}
}

func (tg *TGUI) NotifyEndTurn() {
	tg.notifications <- endTurnNotification{}
}
