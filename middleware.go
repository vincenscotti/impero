package main

import (
	"errors"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	. "impero/model"
	"net/http"
	"net/http/httputil"
	"time"
)

func LoggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := httputil.DumpRequest(r, true)
		logger.Println(string(req))
		next(w, r)
	})
}

func GlobalMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx := db.Begin()

		defer func() {
			errstruct := recover()
			if errstruct != nil {
				tx.Rollback()

				if err, ok := errstruct.(error); ok {
					ErrorHandler(err, w, r)
					return
				} else if err, ok := errstruct.(string); ok {
					ErrorHandler(errors.New(err), w, r)
					return
				} else {
					panic(errstruct)
				}
			} else {
				tx.Commit()
			}
		}()

		opt := &Options{}
		if err := tx.First(opt).Error; err != nil {
			panic(err)
		}

		session, err := store.Get(r, "sess")

		if err != nil {
			panic(err)
		}

		context.Set(r, "now", time.Now())
		context.Set(r, "tx", tx)
		context.Set(r, "options", opt)
		context.Set(r, "session", session)

		next.ServeHTTP(w, r)
	})
}

func GetTime(r *http.Request) time.Time {
	now, ok := context.Get(r, "now").(time.Time)

	if !ok {
		panic("orario non valido")
	}

	return now
}

func GetTx(r *http.Request) *gorm.DB {
	tx, ok := context.Get(r, "tx").(*gorm.DB)

	if !ok {
		panic("transazione non valida")
	}

	return tx
}

func GetOptions(r *http.Request) *Options {
	opt, ok := context.Get(r, "options").(*Options)

	if !ok {
		panic("opzioni di gioco non valide")
	}

	return opt
}

func GetSession(r *http.Request) *sessions.Session {
	session, ok := context.Get(r, "session").(*sessions.Session)

	if !ok {
		panic("sessione non valida")
	}

	return session
}

func HeaderMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx := GetTx(r)
		opt := GetOptions(r)
		session := GetSession(r)
		now := GetTime(r)

		p := &Player{}

		pID, ok := session.Values["playerID"].(uint)

		if !ok {
			url, err := router.Get("home").URL()
			if err != nil {
				panic(err)
			}

			http.Redirect(w, r, url.Path, http.StatusFound)
			return
		}

		if err := tx.Where(pID).First(p).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				url, err := router.Get("logout").URL()
				if err != nil {
					panic(err)
				}

				http.Redirect(w, r, url.Path, http.StatusFound)
				return
			} else {
				panic(err)
			}
		}

		msgs := 0
		if err := tx.Model(&Message{}).Where("`read` = ? and `to_id` = ?", false,
			p.ID).Count(&msgs).Error; err != nil {
			panic(err)
		}

		reports := 0
		if err := tx.Model(&Report{}).Where("`read` = ? and `player_id` = ?", false,
			p.ID).Count(&reports).Error; err != nil {
			panic(err)
		}

		header := &HeaderData{CurrentPlayer: p, NewMessages: msgs, NewReports: reports, Now: now, Options: opt}
		if s := session.Flashes("error_"); len(s) > 0 {
			header.Error = s[0].(string)
		}
		if s := session.Flashes("success_"); len(s) > 0 {
			header.Success = s[0].(string)
		}

		session.Save(r, w)

		context.Set(r, "header", header)

		next.ServeHTTP(w, r)
	})
}

func GameMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GlobalMiddleware(LoggerMiddleware(updateGameStatus(HeaderMiddleware(next)))).ServeHTTP(w, r)
	})
}
