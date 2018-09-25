package main

import (
	"errors"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"time"
)

func Admin(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	err, opt := tx.GetOptions()

	if err != nil {
		panic(err)
	}

	p := &AdminData{Options: &opt}
	if msg := session.Flashes("message_"); len(msg) > 0 {
		p.Message = msg[0].(string)
	}

	RenderHTML(w, r, templates.AdminPage(p))
}

type PasswordForm struct {
	Password string
}

func validateAdmin(r *http.Request) (err error) {
	p := PasswordForm{}

	if err := binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Password != AdminPass {
		err = errors.New("Password errata!")
	}

	return
}

func UpdateOptions(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	newopt := Options{}
	err, oldopt := tx.GetOptions()

	if err != nil {
		panic(err)
	}

	otheropts := struct {
		LastCheckpoint     formTime
		LastTurnCalculated formTime
		Action             string
	}{}

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")
	} else {
		if err := binder.Bind(&newopt, r); err != nil {
			panic(err)
		}

		if err := binder.Bind(&otheropts, r); err != nil {
			panic(err)
		}

		newopt.ID = oldopt.ID
		newopt.LastCheckpoint = time.Time(otheropts.LastCheckpoint)
		newopt.LastTurnCalculated = time.Time(otheropts.LastTurnCalculated)

		if err := tx.SaveOptions(newopt); err != nil {
			panic(err)
		}

		tx.Commit()

		session.AddFlash("Opzioni aggiornate", "message_")
	}

	Redirect(w, r, "admin")
}

var NodeYields = []struct {
	Yield       int
	Prob        float64
	UpgradeCost int
}{
	{1, 0.22, 1},
	{3, 0.5, 2},
	{6, 0.15, 5},
	{12, 0.08, 13},
	{25, 0.04, 30},
	{50, 0.01, 0},
}

func GenerateMap(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	params := struct {
		X0       int
		Y0       int
		X1       int
		Y1       int
		Generate bool
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")
	} else {
		if err := tx.UpdateMapYields(params.X0, params.X1, params.Y0, params.Y1, params.Generate); err != nil {
			session.AddFlash(err.Error(), "message_")
		} else {
			tx.Commit()

			session.AddFlash("Rendimenti aggiornati", "message_")
		}
	}

	Redirect(w, r, "admin")
}

func BroadcastMessage(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	msg := &Message{}

	if err := validateAdmin(r); err != nil {
		session.AddFlash(err.Error(), "message_")

		goto out
	}

	if err := binder.Bind(msg, r); err != nil {
		panic(err)
	}

	if msg.Content == "" {
		session.AddFlash("Non puoi inviare un messaggio vuoto!", "message_")

		goto out
	}

	msg.Date = time.Now()
	msg.Read = false

	if err := tx.BroadcastMessage(msg); err != nil {
		session.AddFlash(err.Error(), "message_")
	} else {
		tx.Commit()

		session.AddFlash("Messaggio inviato!", "message_")
	}

out:
	Redirect(w, r, "admin")
}
