package main

import (
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"golang.org/x/crypto/bcrypt"
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
		cnt := 0
		if err := tx.Model(p).Where(&Player{Name: p.Name}).Count(&cnt).Error; err != nil {
			panic(err)
		}

		if cnt != 0 {
			msg = "Username gia' in uso!"
		} else {
			p.Budget = opt.PlayerBudget
			p.ActionPoints = opt.PlayerActionPoints

			pwdhash, err := bcrypt.GenerateFromPassword([]byte(p.Password), 10)

			if err != nil {
				panic(err)
			}

			p.Password = string(pwdhash)

			if err := tx.Create(&p).Error; err != nil {
				panic(err)
			}

			session.Values["playerID"] = p.ID

			Redirect(w, r, "gamehome")

			return
		}
	}

	RenderHTML(w, r, templates.SignupPage(msg))
}

func Login(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)

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
		hashedp := Player{}
		hashedp.Name = p.Name

		if err := tx.Where(&hashedp).First(&hashedp).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if bcrypt.CompareHashAndPassword([]byte(hashedp.Password), []byte(p.Password)) == nil {
			session.Values["playerID"] = hashedp.ID

			Redirect(w, r, "gamehome")

			return
		} else {
			msg = "Login fallito!"
		}
	}

	RenderHTML(w, r, templates.LoginPage(msg))
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)

	delete(session.Values, "playerID")

	Redirect(w, r, "home")
}
