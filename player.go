package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"strconv"
)

func Players(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	err, players := tx.GetPlayers()

	if err != nil {
		panic(err)
	}

	page := &PlayersData{HeaderData: header, Players: players}

	RenderHTML(w, r, templates.PlayersPage(page))
}

func GetPlayer(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	err, p := tx.GetPlayer(id)

	if err != nil {
		session.AddFlash(err.Error(), "error_")
	}

	page := PlayerData{HeaderData: header, Player: p}

	session.Save(r, w)

	RenderHTML(w, r, templates.PlayerPage(&page))
}

func Transfer(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

	blerr := BLError{}

	params := struct {
		To        string
		Numshares int
		Amount    int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if target, err := router.Get("gamehome").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err, to := tx.GetPlayerByName(params.To); err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		if err, _ := tx.CreateTransferProposal(header.CurrentPlayer, to, params.Amount*100); err != nil {
			session.AddFlash(err.Error(), "error_")
		} else {
			session.AddFlash("Proposta inviata!", "success_")

			tx.Commit()
		}
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func ConfirmTransfer(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := router.Get("gamehome").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	p := &TransferProposal{}
	p.ID = params.ID

	err, fiscalCheck := tx.ConfirmTransferProposal(p)

	if err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		if fiscalCheck {
			session.AddFlash("Controllo fiscale! Il tuo budget e' stato sequestrato!", "warning_")
		} else {
			tx.Commit()

			session.AddFlash("Trasferimento completato!", "success_")
		}
	}

	RedirectToURL(w, r, blerr.Redirect)
}
