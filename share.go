package main

import (
	"fmt"
	"github.com/gorilla/context"
	. "impero/model"
	"net/http"
	"time"
)

func AddShare(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	session := GetSession(r)
	opt := GetOptions(r)
	now := GetTime(r)

	share := &Share{}
	cmp := &Company{}
	if err := tx.Where(params.ID).First(cmp); err.Error != nil {
		panic(err.Error)
	}

	if cmp.CEOID != header.CurrentPlayer.ID {
		session.AddFlash("Permessi insufficienti!", "error_")
		goto out
	}

	if cmp.ActionPoints < 1 {
		session.AddFlash("Punti operazione insufficienti!", "error_")
		goto out
	}

	cmp.ActionPoints -= 1
	if err := tx.Save(cmp); err.Error != nil {
		panic(err.Error)
	}

	share.CompanyID = uint(cmp.ID)
	if err := tx.Create(share); err.Error != nil {
		panic(err.Error)
	}

	if err := tx.Create(&ShareAuction{ShareID: share.ID, HighestOffer: 0, Expiration: now.Add(time.Duration(opt.TurnDuration) * time.Minute)}); err.Error != nil {
		panic(err.Error)
	}

	session.AddFlash("Asta creata", "success_")

out:
	session.Save(r, w)

	url, err := router.Get("company").URL("id", fmt.Sprint(params.ID))
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}

func BidShare(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

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

	// 'record not found' is allowed here
	tx.Where(participation).Find(participation)

	if err := tx.Where(params.Auction).First(shareauction); err.Error != nil {
		panic(err.Error)
	}

	if shareauction.HighestOffer >= params.Amount {
		session.AddFlash("Puntata troppo bassa!", "error_")
		goto out
	}

	if (shareauction.HighestOfferPlayerID != header.CurrentPlayer.ID &&
		params.Amount > header.CurrentPlayer.Budget) ||
		(shareauction.HighestOfferPlayerID == header.CurrentPlayer.ID &&
			params.Amount > header.CurrentPlayer.Budget+
				shareauction.HighestOffer) {
		session.AddFlash("Budget insufficiente!", "error_")
		goto out
	}

	if participation.ID == 0 {
		if header.CurrentPlayer.ActionPoints < 1 {
			session.AddFlash("Punti operazione insufficienti!", "error_")
			goto out
		}

		if err := tx.Save(participation); err.Error != nil {
			panic(err.Error)
		}

		header.CurrentPlayer.ActionPoints -= 1
	}

	header.CurrentPlayer.Budget -= params.Amount
	if err := tx.Save(header.CurrentPlayer); err.Error != nil {
		panic(err.Error)
	}

	// 'record not found' is allowed here
	tx.Where(shareauction.HighestOfferPlayerID).Find(oldp)

	if oldp.ID != 0 {
		oldp.Budget += shareauction.HighestOffer
		if err := tx.Save(oldp); err.Error != nil {
			panic(err.Error)
		}
	}

	shareauction.HighestOffer = params.Amount
	shareauction.HighestOfferPlayerID = header.CurrentPlayer.ID
	if err := tx.Save(shareauction); err.Error != nil {
		panic(err.Error)
	}

	session.AddFlash("Puntata inserita", "success_")

out:
	session.Save(r, w)

	url, err := router.Get("gamehome").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
