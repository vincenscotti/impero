package main

import (
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
)

func Signup(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	tx := GetTx(r)

	p := Player{}
	msg := ""

	if err := binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if r.Method == "POST" {
		err, newp := tx.SignupPlayer(&p)

		if err == nil {
			tx.Commit()

			session.Values["playerID"] = newp.ID

			Redirect(w, r, "gamehome")

			return
		}

		msg = err.Error()
	}

	RenderHTML(w, r, templates.SignupPage(msg))
}

func Login(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	tx := GetTx(r)

	_, ok := session.Values["playerID"].(uint)

	if ok {
		Redirect(w, r, "gamehome")

		return
	}

	p := Player{}
	msg := ""

	if err := binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Name != "" && p.Password != "" {
		err, newp := tx.LoginPlayer(&p)

		if err == nil {
			session.Values["playerID"] = newp.ID

			Redirect(w, r, "gamehome")

			return
		} else {
			msg = err.Error()
		}
	}

	RenderHTML(w, r, templates.LoginPage(msg))
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	delete(session.Values, "playerID")

	Redirect(w, r, "home")
}
