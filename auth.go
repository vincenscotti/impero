package main

import (
	"encoding/json"
	"net/http"

	"github.com/dgrijalva/jwt-go"
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

	_, ok := session.Values["playerID"].(uint)

	if ok {
		s.Redirect(w, r, "gamehome")

		return
	}

	p := Player{}
	msg := ""

	if err := s.binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Name != "" && p.Password != "" {
		err, newp := tx.LoginPlayer(&p)

		if err == nil {
			session.Values["playerID"] = newp.ID

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
		err, _ := tx.LoginPlayer(&p)

		if err == nil {
			mySigningKey := []byte("AllYourBase")

			// Create the Claims
			claims := &jwt.StandardClaims{
				ExpiresAt: 15000,
				Issuer:    "test",
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			ss, _ := token.SignedString(mySigningKey)
			response = struct{ Token string }{ss}
		} else {
			response = err
		}
	}

	RenderJSON(w, r, response)
}

func (s *httpBackend) Logout(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	delete(session.Values, "playerID")

	s.Redirect(w, r, "home")
}
