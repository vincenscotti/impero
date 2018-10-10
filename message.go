package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"strconv"
)

func MessagesInbox(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, msgs := tx.GetInbox(header.CurrentPlayer)

	page := MessagesInboxData{HeaderData: header, Messages: msgs}

	RenderHTML(w, r, templates.MessagesInboxPage(&page))
}

func MessagesOutbox(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, msgs := tx.GetOutbox(header.CurrentPlayer)

	page := MessagesOutboxData{HeaderData: header, Messages: msgs}

	RenderHTML(w, r, templates.MessagesOutboxPage(&page))
}

func GetMessage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := router.Get("message_inbox").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	err, msg := tx.GetMessage(header.CurrentPlayer, id)

	if err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	page := MessageData{HeaderData: header, Message: msg}

	tx.Commit()

	RenderHTML(w, r, templates.MessagePage(&page))
}

func NewMessagePost(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	session := GetSession(r)

	blerr := BLError{}

	if target, err := router.Get("message_outbox").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	msg := &Message{}

	if err := binder.Bind(msg, r); err != nil {
		panic(err)
	}

	to := &Player{}
	to.ID = msg.ToID
	fmt.Println("SENDING TO", to.ID, to, msg.To)

	err := tx.PostMessage(header.CurrentPlayer, to, msg.Subject, msg.Content)

	if err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	session.AddFlash("Messaggio inviato!", "success_")
	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}
