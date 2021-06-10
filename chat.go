package main

import (
	"net/http"

	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
)

func (s *httpBackend) GetChat(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	page := ChatData{HeaderData: header, LastChatViewed: header.CurrentPlayer.LastChatViewed}

	_, msgs := tx.GetChatMessages(header.CurrentPlayer)
	page.Messages = msgs

	RenderHTML(w, r, templates.ChatPage(&page))
}

func (s *httpBackend) PostChat(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := s.router.Get("chat").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	msg := &ChatMessage{}

	if err := s.binder.Bind(msg, r); err != nil {
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
