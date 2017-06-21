package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "impero/model"
	"impero/templates"
	"net/http"
	"strconv"
	"time"
)

func MessagesInbox(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	msgs := make([]*Message, 0)
	if err := tx.Where("`to_id` = ?", header.CurrentPlayer.ID).Preload("From").Order("`Date` desc", true).Find(&msgs); err.Error != nil {
		panic(err.Error)
	}

	page := MessagesInboxData{HeaderData: header, Messages: msgs}

	renderHTML(w, 200, templates.MessagesInboxPage(&page))
}

func MessagesOutbox(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	msgs := make([]*Message, 0)
	if err := tx.Where("`from_id` = ?", header.CurrentPlayer.ID).Preload("To").Order("`Date` desc", true).Find(&msgs); err.Error != nil {
		panic(err.Error)
	}

	page := MessagesOutboxData{HeaderData: header, Messages: msgs}

	renderHTML(w, 200, templates.MessagesOutboxPage(&page))
}

func GetMessage(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	msg := &Message{}
	if err := tx.Preload("From").Preload("To").Where(id).First(msg); err.Error != nil {
		panic(err.Error)
	}

	if msg.FromID != header.CurrentPlayer.ID && msg.ToID != header.CurrentPlayer.ID {
		session.AddFlash("Non puoi leggere questo messaggio!", "error_")

		session.Save(r, w)

		url, err := router.Get("message_inbox").URL()
		if err != nil {
			panic(err)
		}

		http.Redirect(w, r, url.Path, http.StatusFound)
	} else if msg.ToID == header.CurrentPlayer.ID {
		msg.Read = true
		if err := tx.Save(&msg); err.Error != nil {
			panic(err.Error)
		}
	}

	page := MessageData{HeaderData: header, Message: msg}

	renderHTML(w, 200, templates.MessagePage(&page))
}

func NewMessagePost(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	msg := &Message{}

	if err := binder.Bind(msg, r); err != nil {
		panic(err)
	}

	if msg.Content == "" {
		session.AddFlash("Non puoi inviare un messaggio vuoto!", "error_")

		goto out
	}

	if msg.ToID == 0 {
		session.AddFlash("Destinatario non valido!", "error_")

		goto out
	}

	msg.FromID = header.CurrentPlayer.ID
	msg.Date = time.Now()
	msg.Read = false

	if err := tx.Create(msg); err.Error != nil {
		panic(err.Error)
	}

	session.AddFlash("Messaggio inviato!", "success_")

out:
	url, err := router.Get("message_outbox").URL()
	if err != nil {
		panic(err)
	}

	session.Save(r, w)

	http.Redirect(w, r, url.Path, http.StatusFound)
}
