package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "impero/model"
	"impero/templates"
	"math"
	"math/rand"
	"net/http"
	"strconv"
)

func Players(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	players := make([]*Player, 0)
	tx.Order("last_budget desc").Find(&players)
	page := &PlayersData{HeaderData: header, Players: players}

	renderHTML(w, 200, templates.PlayersPage(page))
}

func GetPlayer(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	p := &Player{}
	if err := tx.Where(id).First(p); err.Error != nil {
		panic(err.Error)
	}

	page := PlayerData{HeaderData: header, Player: p}

	renderHTML(w, 200, templates.PlayerPage(&page))
}

func Transfer(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	p := &TransferProposal{}

	if err := binder.Bind(p, r); err != nil {
		panic(err)
	}

	if p.Amount > header.CurrentPlayer.Budget {
		session.AddFlash("Budget insufficiente!", "error_")
		goto out
	}

	if header.CurrentPlayer.ActionPoints < 1 {
		session.AddFlash("Punti operazione insufficienti!", "error_")
		goto out
	}

	p.FromID = header.CurrentPlayer.ID
	p.Risk = int(math.Floor(float64(p.Amount) / float64(header.CurrentPlayer.Budget) * 100))

	header.CurrentPlayer.Budget -= p.Amount
	header.CurrentPlayer.ActionPoints -= 1
	if err := tx.Save(header.CurrentPlayer); err.Error != nil {
		panic(err.Error)
	}

	if err := tx.Create(p); err.Error != nil {
		panic(err.Error)
	}

	session.AddFlash("Proposta inviata!", "success_")

out:
	session.Save(r, w)

	url, err := router.Get("player").URL("id", fmt.Sprint(p.ToID))
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}

func TransferAction(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	p := &TransferProposal{}

	randint := rand.Intn(100) + 1

	if err := tx.Where(params.ID).Preload("From").Preload("To").Find(p); err.Error != nil {
		panic(err.Error)
	}

	if r.FormValue("action") == "Accetta" {
		if header.CurrentPlayer.ActionPoints < 1 {
			session.AddFlash("Punti operazione insufficienti!", "error_")
			goto out
		}

		header.CurrentPlayer.ActionPoints -= 1

		if randint < p.Risk {
			// oops
			header.CurrentPlayer.Budget = 0

			p.From.Budget = 0
			if err := tx.Save(p.From); err.Error != nil {
				panic(err.Error)
			}

			session.AddFlash("CONTROLLO FISCALE! Il tuo budget e' stato sequestrato!", "error_")
		} else {
			// success
			header.CurrentPlayer.Budget += p.Amount

			session.AddFlash("Trasferimento completato!", "success_")
		}

		if err := tx.Save(header.CurrentPlayer); err.Error != nil {
			panic(err.Error)
		}
	} else {
		p.From.Budget += p.Amount
		if err := tx.Save(p.From); err.Error != nil {
			panic(err.Error)
		}

		session.AddFlash("Trasferimento rifiutato!", "success_")
	}

	if err := tx.Delete(p); err.Error != nil {
		panic(err.Error)
	}

out:
	session.Save(r, w)

	url, err := router.Get("gamehome").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}