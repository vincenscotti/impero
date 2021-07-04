package main

import (
	"errors"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/vincenscotti/impero/engine"
	. "github.com/vincenscotti/impero/model"
)

func (s *httpBackend) LoggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := httputil.DumpRequest(r, true)
		s.logger.Println(string(req))
		next(w, r)
	})
}

func (s *httpBackend) APIMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx := s.gameEngine.OpenSession()
		defer tx.Close()

		// TODO: handle panics

		context.Set(r, "tx", tx)
		next.ServeHTTP(w, r)
	})
}

func (s *httpBackend) GlobalMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx := s.gameEngine.OpenSession()
		defer tx.Close()

		var session *sessions.Session
		var err error

		defer func() {
			errstruct := recover()
			if errstruct != nil {
				tx.Rollback()

				if blerr, ok := errstruct.(BLError); ok {
					session.AddFlash(blerr.Message, "error_")
					session.Save(r, w)
					http.Redirect(w, r, blerr.Redirect.Path, http.StatusFound)
				} else if err, ok := errstruct.(error); ok {
					s.ErrorHandler(err, w, r)
				} else if err, ok := errstruct.(string); ok {
					s.ErrorHandler(errors.New(err), w, r)
				} else {
					panic(errstruct)
				}
			} else {
				tx.Commit()
			}
		}()

		if session, err = s.store.Get(r, "sess"); err != nil {
			panic(err)
		}

		context.Set(r, "tx", tx)
		context.Set(r, "session", session)

		next.ServeHTTP(w, r)
	})
}

func GetTx(r *http.Request) *engine.EngineSession {
	tx, ok := context.Get(r, "tx").(*engine.EngineSession)

	if !ok {
		panic("transazione non valida")
	}

	return tx
}

func GetSession(r *http.Request) *sessions.Session {
	session, ok := context.Get(r, "session").(*sessions.Session)

	if !ok {
		panic("sessione non valida")
	}

	return session
}

func (s *httpBackend) HeaderMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx := GetTx(r)
		_, opt := tx.GetOptions()
		session := GetSession(r)
		now := tx.GetTimestamp()

		tokenString, ok := session.Values["tokenString"].(string)

		if !ok {
			s.Redirect(w, r, "home")

			return
		}

		pID, _ := tx.ValidateTokenString(tokenString)
		if pID == nil {
			s.Redirect(w, r, "logout")

			return
		}

		err, p := tx.GetPlayer(int(pID.ID))
		if err != nil {
			panic(err)
		}

		_, chats, msgs, reports := tx.GetPlayerNotifications(int(p.ID))

		header := &HeaderData{CurrentPlayer: p, Router: s.router, NewChatMessages: chats, NewMessages: msgs, NewReports: reports, Now: now, Options: &opt}
		if s := session.Flashes("error_"); len(s) > 0 {
			header.Error = s[0].(string)
		}
		if s := session.Flashes("warning_"); len(s) > 0 {
			header.Warning = s[0].(string)
		}
		if s := session.Flashes("success_"); len(s) > 0 {
			header.Success = s[0].(string)
		}

		session.Save(r, w)

		context.Set(r, "header", header)

		next.ServeHTTP(w, r)
	})
}

func (s *httpBackend) GameMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.LoggerMiddleware(s.GlobalMiddleware(s.HeaderMiddleware(next))).ServeHTTP(w, r)
	})
}
