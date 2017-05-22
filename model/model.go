package model

import (
	"github.com/jinzhu/gorm"
	"time"
)

type Options struct {
	gorm.Model
	LastDividend int
}

type Node struct {
	gorm.Model
	X                int
	Y                int
	ConstructionCost int
	Yield            int
	Owner            Company
	OwnerID          int
}

type Player struct {
	gorm.Model
	Name         string `form:"name"`
	Password     string `form:"password"`
	Budget       int
	ActionPoints int
}

type Message struct {
	gorm.Model
	From    Player
	FromID  int
	To      Player
	ToID    int
	Date    time.Time
	Subject string
	Content string
	Read    bool
}

type Company struct {
	gorm.Model
	Name          string
	ShareCapital  int
	CEO           Player
	CEOID         int
	CEOExpiration int
	ActionPoints  int
}

type Share struct {
	gorm.Model
	Company   Company
	CompanyID int
	Owner     Player
	OwnerID   int
}

type Rental struct {
	gorm.Model
	Node   Node
	NodeID int
	From   Company
	FromID int
	To     Company
	ToID   int
	Price  int
}

type ShareAuction struct {
	gorm.Model
	Share               Share
	ShareID             int
	HighestOffer        int
	HighestOfferPlayer  Player
	HigherOfferPlayerID int
	Expiration          time.Time
}

type RentAuction struct {
	gorm.Model
	Node                  Node
	NodeID                int
	HighestOffer          int
	HighestOfferCompany   Company
	HighestOfferCompanyID int
	Expiration            time.Time
}
