package main

import (
	"encoding/json"
	"net/http"

	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
)

func (s *httpBackend) Signup(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	tx := GetTx(r)

	params := struct {
		Name      string
		Password  string
		Password2 string
	}{}

	p := Player{}
	msg := ""

	if err := s.binder.Bind(&params, r); err != nil {
		panic(err)
	}
	p.Name = params.Name
	p.Password = params.Password

	if r.Method == "POST" {
		if params.Password != params.Password2 {
			msg = "Le password non coincidono!"
		} else {
			err, newp := tx.SignupPlayer(&p)

			if err == nil {
				tx.Commit()

				session.Values["playerID"] = newp.ID

				s.Redirect(w, r, "gamehome")

				return
			}

			msg = err.Error()
		}
	}

	RenderHTML(w, r, templates.SignupPage(msg))
}

func (s *httpBackend) Login(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	tx := GetTx(r)

	tokenString, ok := session.Values["tokenString"].(string)

	if ok {
		if p, _ := tx.ValidateTokenString(tokenString); p != nil {
			s.Redirect(w, r, "gamehome")

			return
		}
	}

	p := Player{}
	msg := ""

	if err := s.binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Name != "" && p.Password != "" {
		err, _, tokenString := tx.LoginPlayer(&p)

		if err == nil {
			session.Values["tokenString"] = tokenString

			s.Redirect(w, r, "gamehome")

			return
		} else {
			msg = err.Error()
		}
	}

	RenderHTML(w, r, templates.LoginPage(msg))
}

func (s *httpBackend) APILogin(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)

	p := Player{}

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid syntax", http.StatusBadRequest)
		return
	}

	var response interface{}

	if p.Name != "" && p.Password != "" {
		err, _, ss := tx.LoginPlayer(&p)

		if err == nil {
			response = struct{ Token string }{ss}
		} else {
			response = err
		}
	}

	RenderJSON(w, r, response)
}

func (s *httpBackend) Logout(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	tx := GetTx(r)

	tokenString, ok := session.Values["tokenString"].(string)

	if ok {
		if _, token := tx.ValidateTokenString(tokenString); token != nil {
			tx.DeleteToken(token)
		}
	}

	delete(session.Values, "tokenString")

	s.Redirect(w, r, "home")
}
