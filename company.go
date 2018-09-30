package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"math"
	"net/http"
	"strconv"
)

func UpdateCompanyIncome(cmp *Company, tx *gorm.DB) {
	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)

	if err := tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
		panic(err)
	}

	income := 0

	for _, n := range nodes {
		income += n.Yield

		if err := tx.Where("`node_id` = ?", n.ID).Find(&rentals).Error; err != nil {
			panic(err.Error())
		}

		for _, _ = range rentals {
			income += int(math.Ceil(float64(n.Yield) / 2.))
		}
	}

	if err := tx.Preload("Node").Where("`tenant_id` = ?", cmp.ID).Find(&rentals).Error; err != nil {
		panic(err)
	}

	for _, r := range rentals {
		income += r.Node.Yield / 2
	}

	cmp.Income = income
}

func GetCompanyPartnerships(cmp *Company, tx *gorm.DB) []*Partnership {
	partnerships := make([]*Partnership, 0)

	tx.Preload("To").Preload("From").Where("`from_id` = ? or `to_id` = ?", cmp.ID, cmp.ID).Find(&partnerships)

	return partnerships
}

func Companies(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	_, companies := tx.GetCompanies()

	page := CompaniesData{HeaderData: header, Companies: companies}

	RenderHTML(w, r, templates.CompaniesPage(&page))
}

func GetCompany(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		panic(err)
	}

	cmp := &Company{}
	shares := 0
	myshares := 0

	if err := tx.Preload("CEO").Where(id).First(cmp).Error; err != nil {
		panic(err)
	}
	if err := tx.Model(&Share{}).Where("`company_id` = ?", id).Where("`owner_id` != 0").Count(&shares).Error; err != nil {
		panic(err)
	}
	if err := tx.Model(&Share{}).Where("`company_id` = ?", id).Where("`owner_id` = ?", header.CurrentPlayer.ID).Count(&myshares).Error; err != nil {
		panic(err)
	}

	UpdateCompanyIncome(cmp, tx)

	floatIncome := float64(cmp.Income)
	pureIncome := math.Floor(floatIncome * 0.3)
	valuepershare := int(math.Ceil((floatIncome - pureIncome) / float64(shares)))

	shareholders := make([]*ShareholdersPerCompany, 0)

	if err := tx.Table("shares").Select("DISTINCT owner_id, count(owner_id) as shares").
		Where("`company_id` = ? and `owner_id` != 0", cmp.ID).
		Group("`owner_id`").Order("`owner_id` asc").
		Find(&shareholders).Error; err != nil {
		panic(err)
	}

	for _, sh := range shareholders {
		if err := tx.Table("players").Where(sh.OwnerID).Find(&sh.Owner).Error; err != nil {
			panic(err)
		}
	}

	ownedcompanies := make([]*Company, 0)
	tx.Where("`ceo_id` = ?", header.CurrentPlayer.ID).Find(&ownedcompanies)

	partnerships := GetCompanyPartnerships(cmp, tx)

	page := CompanyData{HeaderData: header, Company: cmp, SharesInfo: shareholders, Shares: shares, PureIncome: int(pureIncome),
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
