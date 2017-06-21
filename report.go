package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "impero/model"
	"impero/templates"
	"net/http"
	"strconv"
)

func ReportsPage(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	reports := make([]*Report, 0)
	if err := tx.Where("`player_id` = ?", header.CurrentPlayer.ID).Order("`Date` desc", true).Find(&reports).Error; err != nil {
		panic(err)
	}

	page := ReportsData{HeaderData: header, Reports: reports}

	renderHTML(w, 200, templates.ReportsPage(&page))
}

func ReportPage(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	report := &Report{}
	if err := tx.Where(id).First(report).Error; err != nil {
		panic(err)
	}

	if report.PlayerID != header.CurrentPlayer.ID {
		session.AddFlash("Non hai i permessi per vedere questo report!", "error_")

		url, err := router.Get("report_all").URL()
		if err != nil {
			panic(err)
		}

		session.Save(r, w)
		http.Redirect(w, r, url.Path, http.StatusFound)
		return
	}

	report.Read = true
	if err := tx.Save(&report).Error; err != nil {
		panic(err)
	}

	page := ReportData{HeaderData: header, Report: report}

	renderHTML(w, 200, templates.ReportPage(&page))
}

func DeleteReports(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	params := struct {
		IDs []uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	notmine := 0

	if err := tx.Model(&Report{}).Where("`id` in (?) and `player_id` != ?", params.IDs, header.CurrentPlayer.ID).Count(&notmine).Error; err != nil {
		panic(err)
	}

	if notmine > 0 {
		session.AddFlash("Non hai i permessi per cancellare questi report!", "error_")
		goto out
	}

	if err := tx.Delete(&Report{}, "id in (?)", params.IDs).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Report cancellati!", "success_")

out:
	session.Save(r, w)

	url, err := router.Get("report_all").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)

	return
}
