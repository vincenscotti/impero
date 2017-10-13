package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"net/http"
	"time"
)

func AddShare(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	blerr := BLError{}

	params := struct {
		ID     uint
		Amount int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if target, err := router.Get("company").URL("id", fmt.Sprint(params.ID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	session := GetSession(r)
	opt := GetOptions(r)
	now := GetTime(r)

	share := &Share{}
	cmp := &Company{}

	if err := tx.Where(params.ID).First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		blerr.Message = "Societa' inesistente!"
		panic(blerr)
	}

	if cmp.CEOID != header.CurrentPlayer.ID {
		blerr.Message = "Permessi insufficienti!"
		panic(blerr)
	}

	if cmp.ActionPoints < 1 {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	cmp.ActionPoints -= 1
	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	share.CompanyID = uint(cmp.ID)
	if err := tx.Create(share).Error; err != nil {
		panic(err)
	}

	if err := tx.Create(&ShareAuction{ShareID: share.ID, HighestOffer: params.Amount, Expiration: now.Add(time.Duration(opt.TurnDuration) * time.Minute)}).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Asta creata", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}

func BidShare(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	now := GetTime(r)

	blerr := BLError{}

	if target, err := router.Get("gamehome").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := struct {
		Auction uint
		Amount  int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	shareauction := &ShareAuction{}
	oldp := &Player{}
	participation := &ShareAuctionParticipation{}
	participation.ShareAuctionID = params.Auction
	participation.PlayerID = header.CurrentPlayer.ID

	if err := tx.Where(participation).Find(participation).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if err := tx.Where(params.Auction).First(shareauction).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if shareauction.ID == 0 {
		blerr.Message = "L'asta non esiste!"
		panic(blerr)
	}

	if shareauction.HighestOffer >= params.Amount {
		blerr.Message = "Puntata troppo bassa!"
		panic(blerr)
	}

	if (shareauction.HighestOfferPlayerID != header.CurrentPlayer.ID &&
		params.Amount > header.CurrentPlayer.Budget) ||
		(shareauction.HighestOfferPlayerID == header.CurrentPlayer.ID &&
			params.Amount > header.CurrentPlayer.Budget+
				shareauction.HighestOffer) {
		blerr.Message = "Budget insufficiente!"
		panic(blerr)
	}

	if participation.ID == 0 {
		if header.CurrentPlayer.ActionPoints < 1 {
			blerr.Message = "Punti operazione insufficienti!"
			panic(blerr)
		}

		if err := tx.Save(participation).Error; err != nil {
			panic(err)
		}

		header.CurrentPlayer.ActionPoints -= 1
	}

	header.CurrentPlayer.Budget -= params.Amount
	if err := tx.Save(header.CurrentPlayer).Error; err != nil {
		panic(err)
	}

	if err := tx.Where(shareauction.HighestOfferPlayerID).Find(oldp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if oldp.ID != 0 {
		oldp.Budget += shareauction.HighestOffer
		if err := tx.Save(oldp).Error; err != nil {
			panic(err)
		}
	}

	shareauction.HighestOffer = params.Amount
	shareauction.HighestOfferPlayerID = header.CurrentPlayer.ID

	if shareauction.Expiration.Sub(now).Minutes() < 1. {
		shareauction.Expiration = now.Add(time.Minute)
	}

	if err := tx.Save(shareauction).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Puntata inserita", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}
