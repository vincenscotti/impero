package main

import (
	"fmt"
	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"net/http"
)

func EmitShares(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	params := struct {
		ID        uint
		Numshares int
		Price     int
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

	cmp := &Company{}
	cmp.ID = params.ID

	err := tx.CreateAuction(header.CurrentPlayer, cmp, params.Numshares, params.Price*100)

	if err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		tx.Commit()

		session.AddFlash("Asta creata", "success_")
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func BidShare(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

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
	shareauction.ID = params.Auction

	err := tx.BidAuction(header.CurrentPlayer, shareauction, params.Amount*100)
	if err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		tx.Commit()

		session.AddFlash("Puntata inserita", "success_")
	}

	RedirectToURL(w, r, blerr.Redirect)
}
