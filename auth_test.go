package main

import (
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vincenscotti/impero/engine"
)

const testAdminPassword = "4dm1npwd"
const testBaseURL = "http://example.com/"

func getTestServer() (httpBackend, *gorm.DB) {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	db.LogMode(true)

	logger := log.New(os.Stdout, "impero: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	gameEngine := engine.NewEngine(db, logger, defaultTimeProvider{}, []byte("testJwtPassword"))
	gameEngine.Boot()

	return newHttpBackend(gameEngine, logger, testAdminPassword, true), db
}

func closeTestServer(s *httpBackend, db *gorm.DB) {
	db.Close()
}

func getURLForRoute(s *httpBackend, route string) string {
	endpoint, err := s.router.Get(route).URL()
	if err != nil {
		panic(err)
	}
	return testBaseURL + endpoint.String()
}

func TestAPILogin(t *testing.T) {
	s, db := getTestServer()
	defer closeTestServer(&s, db)

	url := getURLForRoute(&s, "api_login")
	req := httptest.NewRequest("POST", url, nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))
}
