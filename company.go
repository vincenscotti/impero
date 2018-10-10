package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
	"strconv"
)

func Companies(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	_, companies := tx.GetCompanies()

	page := CompaniesData{HeaderData: header, Companies: companies}

	RenderHTML(w, r, templates.CompaniesPage(&page))
}

func GetCompany(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	_, cmp, shareholders, pureincome, valuepershare := tx.GetCompany(header.CurrentPlayer, id)
	_, partnerships := tx.GetCompanyPartnerships(cmp)
	_, ownedcompanies := tx.GetOwnedCompanies(header.CurrentPlayer)

	shares := 0
	myshares := 0

	for _, shs := range shareholders {
		shares += shs.Shares

		if shs.OwnerID == header.CurrentPlayer.ID {
			myshares = shs.Shares
		}
	}

	page := CompanyData{HeaderData: header, Company: cmp, SharesInfo: shareholders, Shares: shares, PureIncome: pureincome,
		IncomePerShare: valuepershare, IsShareHolder: myshares >= 1, PossiblePartners: ownedcompanies, Partnerships: partnerships}

	RenderHTML(w, r, templates.CompanyPage(&page))
}

func NewCompanyPost(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	if err := tx.NewCompany(header.CurrentPlayer, cmp.Name, cmp.ShareCapital); err != nil {
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

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	session.AddFlash("Sei diventato amministratore!", "error_")
	tx.Commit()

	RedirectToURL(w, r, blerr.Redirect)
}

func ProposePartnership(w http.ResponseWriter, r *http.Request) {
	session := GetSession(r)
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	tx := gameEngine.OpenSession()
	defer tx.Close()

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
