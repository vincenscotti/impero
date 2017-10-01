package main

import (
	"github.com/gorilla/schema"
	"net/http"
	"time"
)

type gorillaBinder struct {
	decoder *schema.Decoder
}

func NewGorillaBinder() *gorillaBinder {
	formDecoder := schema.NewDecoder()
	formDecoder.IgnoreUnknownKeys(true)

	return &gorillaBinder{formDecoder}
}

func (this *gorillaBinder) Bind(i interface{}, r *http.Request) error {
	err := r.ParseForm()

	if err != nil {
		return err
	}

	return this.decoder.Decode(i, r.PostForm)
}

var binder *gorillaBinder

type formTime time.Time

func (this *formTime) UnmarshalText(text []byte) error {
	t, err := time.Parse("2006-01-02 15:04:05-07:00", string(text))
	*this = formTime(t)

	return err
}

func SaveSession(w http.ResponseWriter, r *http.Request) {
	GetSession(r).Save(r, w)
}

func RenderHTML(w http.ResponseWriter, r *http.Request, s string) (err error) {
	SaveSession(w, r)

	w.Header().Set("Content-type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	_, err = w.Write([]byte(s))
	return
}

func Redirect(w http.ResponseWriter, r *http.Request, to string) {
	SaveSession(w, r)

	url, err := router.Get(to).URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
