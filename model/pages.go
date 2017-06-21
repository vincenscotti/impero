package model

import "time"

type AdminData struct {
	Options *Options
	Message string
}

type HeaderData struct {
	CurrentPlayer *Player
	Error         string
	Success       string
	NewMessages   int
	NewReports    int
	Now           time.Time
	Options       *Options
}

type GameHomeData struct {
	*HeaderData
	SharesInfo        []*SharesPerPlayer
	ShareAuctions     []*ShareAuction
	IncomingTransfers []*TransferProposal
}

type SharesPerPlayer struct {
	Company   Company
	CompanyID uint
	Shares    int
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
	Company        *Company
	Income         int
	Shares         int
	PureIncome     int
	IncomePerShare int
	SharesInfo     []*ShareholdersPerCompany
	CanVote        bool
	VotedFor       int
}

type ShareholdersPerCompany struct {
	Owner    Player
	OwnerID  uint
	Shares   int
	VotedFor uint
}

type ElectionResults struct {
	ShareHolderID uint
	Shares        int
	Votes         int
}

type MapData struct {
	*HeaderData
	Nodes           map[Point]*Node
	Rentals         []*Rental
	CompaniesByName map[string]*Company
	XMin            int
	YMin            int
	XMax            int
	YMax            int
}
