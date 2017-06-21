package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	. "impero/model"
	"impero/templates"
	"math"
	"math/rand"
	"net/http"
	"strconv"
)

func Companies(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	companies := make([]*Company, 0)
	if err := tx.Order("share_capital desc").Find(&companies).Error; err != nil {
		panic(err)
	}

	for _, cmp := range companies {
		nodes := make([]*Node, 0)
		rentals := make([]*Rental, 0)

		if err := tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
			panic(err)
		}

		income := 0
		for _, n := range nodes {
			income += n.Yield
		}

		if err := tx.Preload("Node").Where("`tenant_id` = ?", cmp.ID).Find(&rentals); err != nil {
			panic(err)
		}

		for _, r := range rentals {
			income += r.Node.Yield / 2
		}

		cmp.Income = income
	}

	page := CompaniesData{HeaderData: header, Companies: companies}

	renderHTML(w, 200, templates.CompaniesPage(&page))
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
	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)
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

	// calcolo income

	income := 0

	if err := tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
		panic(err)
	}

	for _, n := range nodes {
		income += n.Yield
	}

	if err := tx.Preload("Node").Where("`tenant_id` = ?", cmp.ID).Find(&rentals).Error; err != nil {
		panic(err)
	}

	for _, r := range rentals {
		income += r.Node.Yield / 2
	}

	floatIncome := float64(income)
	pureIncome := math.Floor(floatIncome * 0.3)
	valuepershare := int(math.Ceil((floatIncome - pureIncome) / float64(shares)))

	shareholders := make([]*ShareholdersPerCompany, 0)

	if err := tx.Table("shares").Select("DISTINCT owner_id, count(owner_id) as shares").Where("`company_id` = ? and `owner_id` != 0", cmp.ID).Group("`owner_id`").Order("`owner_id` asc").Find(&shareholders).Error; err != nil {
		panic(err)
	}

	for _, sh := range shareholders {
		if err := tx.Table("players").Where(sh.OwnerID).Find(&sh.Owner).Error; err != nil {
			panic(err)
		}
	}

	page := CompanyData{HeaderData: header, Company: cmp, Income: income, SharesInfo: shareholders, Shares: shares, PureIncome: int(pureIncome), IncomePerShare: valuepershare, IsShareHolder: myshares >= 1}

	renderHTML(w, 200, templates.CompanyPage(&page))
}

func NewCompanyPost(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	opt := GetOptions(r)

	cmp := &Company{}
	freenodes := make([]*Node, 0)
	cnt := 0

	if err := binder.Bind(cmp, r); err != nil {
		panic(err)
	}

	if cmp.Name == "" {
		session.AddFlash("Il nome non puo' essere vuoto!", "error_")
		goto out
	}

	if cmp.ShareCapital < 1 {
		session.AddFlash("Il budget deve essere almeno 1!", "error_")
		goto out
	}

	if cmp.ShareCapital > header.CurrentPlayer.Budget {
		session.AddFlash("Budget insufficiente!", "error_")
		goto out
	}

	if header.CurrentPlayer.ActionPoints < opt.NewCompanyCost {
		session.AddFlash("Punti operazione insufficienti!", "error_")
		goto out
	}

	if err := tx.Model(cmp).Where(&Company{Name: cmp.Name}).Count(&cnt).Error; err != nil {
		panic(err)
	}

	if cnt != 0 {
		session.AddFlash("Societa' gia' esistente!", "error_")
		goto out
	}

	header.CurrentPlayer.Budget -= cmp.ShareCapital
	header.CurrentPlayer.ActionPoints -= opt.NewCompanyCost
	cmp.CEO = *header.CurrentPlayer
	cmp.ActionPoints = opt.CompanyActionPoints

	if err := tx.Create(cmp).Error; err != nil {
		panic(err)
	}

	if err := tx.Save(header.CurrentPlayer).Error; err != nil {
		panic(err)
	}

	if err := tx.Create(&Share{CompanyID: cmp.ID, OwnerID: header.CurrentPlayer.ID}).Error; err != nil {
		panic(err)
	}

	if err := tx.Where("`owner_id` = 0 and `yield` = 1").Find(&freenodes).Error; err != nil {
		panic(err)
	}

	if len(freenodes) != 0 {
		node := freenodes[rand.Intn(len(freenodes))]

		node.OwnerID = cmp.ID

		if err := tx.Save(node).Error; err != nil {
			panic(err)
		}

		session.AddFlash("Societa' creata", "success_")
	} else {
		session.AddFlash("Nessuna cella disponibile!", "error_")

		tx.Rollback()
	}

out:
	session.Save(r, w)

	url, err := router.Get("gamehome").URL()
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
