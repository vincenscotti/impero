package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"strconv"
)

func Players(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	tx := gameEngine.OpenSession()
	defer tx.Close()

	blerr := BLError{}

	p := &TransferProposal{}

	if err := binder.Bind(p, r); err != nil {
		panic(err)
	}

	if target, err := router.Get("player").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	to := &Player{}
	to.ID = p.ToID

	if err, _ := tx.CreateTransferProposal(header.CurrentPlayer, to, p.Amount); err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		session.AddFlash("Proposta inviata!", "success_")

		tx.Commit()
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func ConfirmTransfer(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

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
			session.AddFlash("CONTROLLO FISCALE! Il tuo budget e' stato sequestrato!", "error_")
		} else {
			tx.Commit()

			session.AddFlash("Trasferimento completato!", "success_")
		}
	}

	RedirectToURL(w, r, blerr.Redirect)
}
