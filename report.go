package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"strconv"
)

func ReportsPage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, reports := tx.GetReports(header.CurrentPlayer)

	page := ReportsData{HeaderData: header, Reports: reports}

	RenderHTML(w, r, templates.ReportsPage(&page))
}

func ReportPage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := router.Get("report_all").URL(); err != nil {
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

func DeleteReports(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	session := GetSession(r)

	blerr := BLError{}

	if target, err := router.Get("report_all").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	params := struct {
		IDs []int
	}{}

	if err := binder.Bind(&params, r); err != nil {
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
