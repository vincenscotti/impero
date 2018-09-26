package main

import (
	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
)

func GetChat(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	_, msgs := tx.GetChatMessages(header.CurrentPlayer)

	page := ChatData{HeaderData: header, Messages: msgs}

	RenderHTML(w, r, templates.ChatPage(&page))
}

func PostChat(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	blerr := BLError{}

	if target, err := router.Get("chat").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	msg := &ChatMessage{}

	if err := binder.Bind(msg, r); err != nil {
		panic(err)
	}

	err := tx.PostChatMessage(header.CurrentPlayer, msg.Content)

	if err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}
