package main

import (
	"net/http"
	"strconv"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
)

func (s *httpBackend) ReportsPage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, reports := tx.GetReports(header.CurrentPlayer)

	page := ReportsData{HeaderData: header, Reports: reports}

	RenderHTML(w, r, templates.ReportsPage(&page))
}

func (s *httpBackend) ReportPage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := s.router.Get("report_all").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	err, report := tx.GetReport(header.CurrentPlayer, id)

	if err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	page := ReportData{HeaderData: header, Report: report}

	tx.Commit()

	RenderHTML(w, r, templates.ReportPage(&page))
}

func (s *httpBackend) DeleteReports(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	session := GetSession(r)

	blerr := BLError{}

	if target, err := s.router.Get("report_all").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := struct {
		IDs []int
	}{}

	if err := s.binder.Bind(&params, r); err != nil {
		panic(err)
	}

	err := tx.DeleteReports(header.CurrentPlayer, params.IDs)

	if err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	session.AddFlash("Report cancellati!", "success_")

	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}
