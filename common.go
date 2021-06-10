package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/schema"
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

func RenderJSON(w http.ResponseWriter, r *http.Request, obj interface{}) (err error) {
	SaveSession(w, r)

	ret, err := json.Marshal(obj)

	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	_, err = w.Write(ret)

	return
}

func (s *httpBackend) Redirect(w http.ResponseWriter, r *http.Request, to string) {
	u, err := s.router.Get(to).URL()
	if err != nil {
		panic(err)
	}

	RedirectToURL(w, r, u)
}

func RedirectToURL(w http.ResponseWriter, r *http.Request, to *url.URL) {
	SaveSession(w, r)

	http.Redirect(w, r, to.Path, http.StatusFound)
}
