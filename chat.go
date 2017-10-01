package main

import (
	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"time"
)

func GetChat(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	msgs := make([]*ChatMessage, 0)

	if err := tx.Preload("From").Order("Date desc", true).Find(&msgs).Error; err != nil {
		panic(err)
	}

	header.CurrentPlayer.LastChatViewed = GetTime(r)
	if err := tx.Save(header.CurrentPlayer).Error; err != nil {
		panic(err)
	}

	page := ChatData{HeaderData: header, Messages: msgs}

	RenderHTML(w, r, templates.ChatPage(&page))
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
	} else {
		msg.FromID = header.CurrentPlayer.ID
		msg.Date = time.Now()

		if err := tx.Create(msg).Error; err != nil {
			panic(err)
		}

	}

	Redirect(w, r, "chat")
}
