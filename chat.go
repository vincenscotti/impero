package main

import (
	"github.com/gorilla/context"
	. "impero/model"
	"impero/templates"
	"net/http"
	"time"
)

func GetChat(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	msgs := make([]*ChatMessage, 0)
	tx.Preload("From").Order("Date desc", true).Find(&msgs)
	page := ChatData{HeaderData: header, Messages: msgs}

	renderHTML(w, 200, templates.ChatPage(&page))
}

func PostChat(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	msg := &ChatMessage{}

	if err := binder.Bind(msg, r); err != nil {
		panic(err)
	}

	if msg.Content == "" {
		session.AddFlash("Messaggio vuoto non valido!", "error_")

		session.Save(r, w)
	} else {
		msg.FromID = header.CurrentPlayer.ID
		msg.Date = time.Now()

		if err := tx.Create(msg); err.Error != nil {
			panic(err.Error)
		}

	}

	url, err := router.Get("chat").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
