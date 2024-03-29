package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"sort"
	"strconv"
)

func Companies(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, companies := tx.GetCompanies()

	page := CompaniesData{HeaderData: header, Companies: companies}

	RenderHTML(w, r, templates.CompaniesPage(&page))
}

func Stats(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, companies := tx.GetCompanies()
	_, players := tx.GetPlayers()
	sort.Sort(CompaniesSortableByIncome(companies))

	page := StatsData{HeaderData: header, Companies: companies, Players: players}

	RenderHTML(w, r, templates.StatsPage(&page))
}

func GetCompany(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	_, cmp, pureincome, valuepershare := tx.GetCompany(id)
	_, partnerships := tx.GetCompanyPartnerships(cmp)
	_, ownedcompanies := tx.GetOwnedCompanies(header.CurrentPlayer)

	shares := 0
	myshares := 0

	for _, shs := range cmp.Shareholders {
		shares += shs.Shares

		if shs.PlayerID == header.CurrentPlayer.ID {
			myshares = shs.Shares
		}
	}

	page := CompanyData{HeaderData: header, Company: cmp, Shares: shares, PureIncome: pureincome,
		IncomePerShare: valuepershare, IsShareHolder: myshares >= 1, PossiblePartners: ownedcompanies, Partnerships: partnerships}

	RenderHTML(w, r, templates.CompanyPage(&page))
}

func NewCompanyPost(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

	blerr := BLError{}

	if target, err := router.Get("gamehome").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	cmp := &Company{}

	if err := binder.Bind(cmp, r); err != nil {
		panic(err)
	}

	if err := tx.NewCompany(header.CurrentPlayer, cmp.Name, cmp.ShareCapital*100); err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	} else {
		session.AddFlash("Societa' creata", "success_")
		tx.Commit()
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func PromoteCEO(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	tx := GetTx(r)

	blerr := BLError{}

	cmp := &Company{}
	newceo := &Player{}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	cmp.ID = params.ID
	newceo.ID = header.CurrentPlayer.ID

	if target, err := router.Get("company").URL("id", fmt.Sprint(params.ID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err := tx.PromoteCEO(cmp, newceo); err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	session.AddFlash("Sei diventato amministratore!", "success_")
	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}

func ProposePartnership(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	p := &Partnership{}
	from := &Company{}
	to := &Company{}

	if err := binder.Bind(p, r); err != nil {
		panic(err)
	}

	from.ID = p.FromID
	to.ID = p.ToID

	if target, err := router.Get("company").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err := tx.ProposePartnership(header.CurrentPlayer, from, to); err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	} else {
		session.AddFlash("Proposa inviata!", "success_")
		tx.Commit()
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func ConfirmPartnership(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	p := &Partnership{}

	p.ID = params.ID

	if target, err := router.Get("company").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err := tx.ConfirmPartnership(header.CurrentPlayer, p); err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	} else {
		session.AddFlash("Partnership confermata!", "success_")
		tx.Commit()
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func DeletePartnership(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	p := &Partnership{}
	p.ID = params.ID

	if target, err := router.Get("company").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err := tx.DeletePartnership(header.CurrentPlayer, p); err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	session.AddFlash("Partnership cancellata!", "success_")
	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}

func ModifyCompanyPureIncome(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	blerr := BLError{}

	cmp := &Company{}

	params := struct {
		ID     uint
		Action string
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	cmp.ID = params.ID

	if target, err := router.Get("company").URL("id", fmt.Sprint(cmp.ID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	var err error

	if params.Action != "inc" && params.Action != "dec" {
		blerr.Message = "Azione non riconosciuta!"
		panic(blerr)
	}

	if err = tx.ModifyCompanyPureIncome(header.CurrentPlayer, cmp, params.Action == "inc"); err != nil {
		blerr.Message = err.Error()
		panic(blerr)
	}

	session.AddFlash("Percentuale modificata!", "success_")
	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}
