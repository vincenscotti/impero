package model

import (
	"github.com/gorilla/mux"
	"time"
)

type AdminData struct {
	Router  *mux.Router
	Options *Options
	Message string
}

type HeaderData struct {
	CurrentPlayer   *Player
	Router          *mux.Router
	Error           string
	Success         string
	NewChatMessages int
	NewMessages     int
	NewReports      int
	Now             time.Time
	Options         *Options
}

type GameHomeData struct {
	*HeaderData
	SharesInfo        []*SharesPerPlayer
	PlayerIncome      int
	ShareAuctions     []*ShareAuction
	ShareOffers       []*ShareOffer
	IncomingTransfers []*TransferProposal
}

type EndGameData struct {
	*HeaderData
	Players []*Player
	Winners []*Player
}

type SharesPerPlayer struct {
	Company       Company
	CompanyID     uint
	Shares        int
	ValuePerShare int
}

type PlayersData struct {
	*HeaderData
	Players []*Player
}

type MessagesInboxData struct {
	*HeaderData
	Messages []*Message
}

type MessagesOutboxData struct {
	*HeaderData
	Messages []*Message
}

type MessageData struct {
	*HeaderData
	Message *Message
}

type ReportsData struct {
	*HeaderData
	Reports []*Report
}

type ReportData struct {
	*HeaderData
	Report *Report
}

type ChatData struct {
	*HeaderData
	Messages []*ChatMessage
}

type PlayerData struct {
	*HeaderData
	Player *Player
}

type CompaniesData struct {
	*HeaderData
	Companies []*Company
}

type CompanyData struct {
	*HeaderData
	Company          *Company
	Shares           int
	PureIncome       int
	IncomePerShare   int
	SharesInfo       []*ShareholdersPerCompany
	IsShareHolder    bool
	PossiblePartners []*Company
	Partnerships     []*Partnership
}

type ShareholdersPerCompany struct {
	Owner   Player
	OwnerID uint
	Shares  int
}

type MapData struct {
	*HeaderData
	Nodes           map[Coord]*Node
	Rentals         []*Rental
	CompaniesByName map[string]*Company
	XMin            int
	YMin            int
	XMax            int
	YMax            int
}
