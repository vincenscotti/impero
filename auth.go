package main

import (
	"golang.org/x/crypto/bcrypt"
	. "impero/model"
	"impero/templates"
	"net/http"
)

func Signup(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)
	opt := GetOptions(r)

	p := Player{}
	msg := ""

	if err := binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Name != "" && p.Password != "" {
		p.Budget = opt.PlayerBudget
		p.ActionPoints = opt.PlayerActionPoints

		cnt := 0
		if err := tx.Model(p).Where(&Player{Name: p.Name}).Count(&cnt).Error; err != nil {
			panic(err)
		}

		if cnt != 0 {
			msg = "Username gia' in uso!"
		} else {
			pwdhash, err := bcrypt.GenerateFromPassword([]byte(p.Password), 10)

			if err != nil {
				panic(err)
			}

			p.Password = string(pwdhash)

			if err := tx.Create(&p).Error; err != nil {
				panic(err)
			}

			session.Values["playerID"] = p.ID
			session.Save(r, w)

			url, err := router.Get("gamehome").URL()
			if err != nil {
				panic(err)
			}

			http.Redirect(w, r, url.Path, http.StatusFound)

			return
		}
	}

	renderHTML(w, 200, templates.SignupPage(msg))
}

func Login(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)

	_, ok := session.Values["playerID"].(uint)

	if ok {
		url, err := router.Get("gamehome").URL()
		if err != nil {
			panic(err)
		}

		http.Redirect(w, r, url.Path, http.StatusFound)

		return
	}

	p := Player{}
	msg := ""

	if err := binder.Bind(&p, r); err != nil {
		panic(err)
	}

	if p.Name != "" && p.Password != "" {
		hashedp := Player{}
		hashedp.Name = p.Name

		if err := tx.Where(&hashedp).FirstOrInit(&hashedp, hashedp).Error; err != nil {
			panic(err)
		}

		if hashedp.ID != 0 &&
			bcrypt.CompareHashAndPassword([]byte(hashedp.Password), []byte(p.Password)) == nil {
			session.Values["playerID"] = hashedp.ID
			session.Save(r, w)

			url, err := router.Get("gamehome").URL()
			if err != nil {
				panic(err)
			}

			http.Redirect(w, r, url.Path, http.StatusFound)

			return
		} else {
			msg = "Login fallito!"
		}
	}

	renderHTML(w, 200, templates.LoginPage(msg))
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	delete(session.Values, "playerID")
	session.Save(r, w)

	url, err := router.Get("home").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
