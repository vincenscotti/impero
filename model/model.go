package model

import (
	"github.com/jinzhu/gorm"
	"time"
)

type Options struct {
	ID                          uint
	LastTurnCalculated          time.Time `schema:"-"`
	TurnDuration                int
	PlayerActionPoints          int
	CompanyActionPoints         int
	CompanyPureIncomePercentage int
	PlayerBudget                int
	NewCompanyCost              int
	InitialShares               int
	CostPerYield                float64
	Turn                        int
	EndGame                     int
}

type Coord struct {
	X int
	Y int
}

const (
	PowerOK = iota
	PowerOff
	PowerOffNeighbour
)

type Node struct {
	gorm.Model
	X           int
	Y           int
	Yield       int
	PowerSupply int
	Owner       Company
	OwnerID     uint
}

type Player struct {
	gorm.Model
	Name           string
	Password       string
	Budget         int
	LastBudget     int
	ActionPoints   int
	VP             int
	LastIncome     int
	LastChatViewed time.Time
}

type Message struct {
	gorm.Model
	From    Player
	FromID  uint
	To      Player
	ToID    uint `schema:"to_id"`
	Date    time.Time
	Subject string
	Content string `gorm:"type:text"`
	Read    bool
}

type Report struct {
	gorm.Model
	Player   Player
	PlayerID uint
	Date     time.Time
	Subject  string
	Content  string `gorm:"type:text"`
	Read     bool
}

type ChatMessage struct {
	gorm.Model
	From    Player
	FromID  uint
	Date    time.Time
	Content string
}

type Company struct {
	gorm.Model
	Name                 string `gorm:"size:30"`
	ShareCapital         int
	CEO                  Player
	CEOID                uint
	ActionPoints         int
	Income               int `gorm:"-"`
	PureIncomePercentage int
	Color                int32 `gorm:"-"`
}

type Partnership struct {
	gorm.Model
	From               Company
	FromID             uint
	To                 Company
	ToID               uint
	ProposalAccepted   bool
	ProposalExpiration time.Time
}

type Share struct {
	gorm.Model
	Company   Company
	CompanyID uint
	Owner     Player
	OwnerID   uint
}

type ShareAuction struct {
	gorm.Model
	Share                Share
	ShareID              uint
	HighestOffer         int
	HighestOfferPlayer   Player
	HighestOfferPlayerID uint
	Expiration           time.Time
	Participations       []*ShareAuctionParticipation `gorm:"ForeignKey:ShareAuctionID"`
}

type ShareAuctionParticipation struct {
	gorm.Model
	ShareAuction   ShareAuction
	ShareAuctionID uint
	Player         Player
	PlayerID       uint
}

type Rental struct {
	gorm.Model
	Node     Node
	NodeID   uint
	Tenant   Company
	TenantID uint
}

type TransferProposal struct {
	gorm.Model
	From       Player
	FromID     uint
	To         Player
	ToID       uint `schema:"to_id"`
	Amount     int
	Risk       int
	Expiration time.Time
}
