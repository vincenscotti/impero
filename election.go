package main

import (
	"fmt"
	"github.com/gorilla/context"
	. "impero/model"
	"net/http"
)

func NewElectionProposal(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	ep := &ElectionProposal{}

	params := struct {
		ID     uint
		Delete bool
		Text   string
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	cnt := 0
	if err := db.Model(&Share{}).Where("owner_id = ? and company_id = ?", header.CurrentPlayer.ID, params.ID).Count(&cnt); err.Error != nil {
		panic(err.Error)
	}

	if cnt == 0 {
		session.AddFlash("Non puoi candidarti in questa societa'!", "error_")

		goto out
	}

	ep.PlayerID = header.CurrentPlayer.ID
	ep.CompanyID = uint(params.ID)

	// 'record not found' allowed
	tx.Where(ep).Find(ep)

	if params.Delete {
		if ep.ID != 0 {
			if err := tx.Delete(ep); err.Error != nil {
				panic(err.Error)
			}

			if err := tx.Delete(&ElectionVote{}, "to_id = ?", header.CurrentPlayer.ID); err.Error != nil {
				panic(err.Error)
			}

			session.AddFlash("Proposta cancellata!", "success_")
		}
	} else {
		ep.Text = params.Text

		if err := tx.Save(ep); err.Error != nil {
			panic(err.Error)
		}

		session.AddFlash("Proposta inserita!", "success_")
	}

out:
	session.Save(r, w)

	url, err := router.Get("company").URL("id", fmt.Sprint(params.ID))
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}

func SetElectionVote(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	ev := &ElectionVote{}

	params := struct {
		ID   uint
		Vote uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	cnt := 0
	if err := db.Model(&Share{}).Where("owner_id = ? and company_id = ?", header.CurrentPlayer.ID, params.ID).Count(&cnt); err.Error != nil {
		panic(err.Error)
	}

	if cnt == 0 {
		session.AddFlash("Non puoi votare in questa societa'!", "error_")

		goto out
	}

	if err := db.Model(&ElectionProposal{}).Where("player_id = ? and company_id = ?", params.Vote, params.ID).Count(&cnt); err.Error != nil {
		panic(err.Error)
	}

	if cnt == 0 {
		session.AddFlash("Il giocatore non e' candidato!", "error_")

		goto out
	}

	ev.CompanyID = params.ID
	ev.FromID = uint(header.CurrentPlayer.ID)

	// 'record not found' allowed
	tx.Where(ev).Find(ev)

	ev.ToID = params.Vote
	if err := tx.Save(ev); err.Error != nil {
		panic(err.Error)
	}

	session.AddFlash("Voto inserito!", "success_")

out:
	session.Save(r, w)

	url, err := router.Get("company").URL("id", fmt.Sprint(params.ID))
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
