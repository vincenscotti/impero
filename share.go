package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
)

func (s *httpBackend) Market(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, shareauctions := tx.GetShareAuctionsWithPlayerParticipation(header.CurrentPlayer)
	_, shareoffers := tx.GetShareOffers()

	page := &MarketData{HeaderData: header,
		ShareAuctions: shareauctions, ShareOffers: shareoffers}

	RenderHTML(w, r, templates.MarketPage(page))
}

func (s *httpBackend) EmitShares(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	params := struct {
		ID        uint
		Numshares int
		Price     int
	}{}

	if err := s.binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if target, err := s.router.Get("company").URL("id", fmt.Sprint(params.ID)); err != nil {
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

func (s *httpBackend) SellShares(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	params := struct {
		ID        uint
		Numshares int
		Price     int
	}{}

	if err := s.binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if target, err := s.router.Get("company").URL("id", fmt.Sprint(params.ID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	session := GetSession(r)

	cmp := &Company{}
	cmp.ID = params.ID

	err := tx.SellShares(header.CurrentPlayer, cmp, params.Numshares, params.Price*100)

	if err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		tx.Commit()

		session.AddFlash("Proposta di vendita registrata", "success_")
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func (s *httpBackend) BidShare(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := s.router.Get("market").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := struct {
		Auction uint
		Amount  int
	}{}

	if err := s.binder.Bind(&params, r); err != nil {
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

func (s *httpBackend) BuyShare(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := s.router.Get("market").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := struct {
		Offer uint
	}{}

	if err := s.binder.Bind(&params, r); err != nil {
		panic(err)
	}

	shareoffer := &ShareOffer{}
	shareoffer.ID = params.Offer

	err := tx.BuyShare(header.CurrentPlayer, shareoffer)
	if err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		tx.Commit()

		session.AddFlash("Azione acquistata", "success_")
	}

	RedirectToURL(w, r, blerr.Redirect)
}
