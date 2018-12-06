package tgui

import (
	. "github.com/vincenscotti/impero/model"
)

type gameNotification interface {
}

type endTurnNotification struct {
}

type auctionRaiseNotification struct {
	company *Company
	auction *ShareAuction
	players []*Player
}

type auctionEndNotification struct {
	company *Company
	auction *ShareAuction
	players []*Player
}

func (tg *TGUI) NotifyAuctionRaise(company *Company, auction *ShareAuction, players []*Player) {
	tg.notifications <- auctionRaiseNotification{company: company, auction: auction, players: players}
}

func (tg *TGUI) NotifyAuctionEnd(company *Company, auction *ShareAuction, players []*Player) {
	tg.notifications <- auctionEndNotification{company: company, auction: auction, players: players}
}

func (tg *TGUI) NotifyEndTurn() {
	tg.notifications <- endTurnNotification{}
}
