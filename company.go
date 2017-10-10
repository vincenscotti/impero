package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"math"
	"math/rand"
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

func GetCompanyFinancials(cmp *Company, tx *gorm.DB) (pureIncome, valuePerShare int) {
	if cmp.Income == 0 {
		UpdateCompanyIncome(cmp, tx)
	}

	cmpshares := 0

	if err := tx.Table("shares").Where("`company_id` = ?", cmp.ID).Where("`owner_id` != 0").Count(&cmpshares).Error; err != nil {
		panic(err)
	}

	floatIncome := float64(cmp.Income)
	floatPureIncome := math.Floor(floatIncome * 0.3)
	floatValuePerShare := int(math.Ceil((floatIncome - floatPureIncome) / float64(cmpshares)))

	return int(floatPureIncome), int(floatValuePerShare)
}

func GetCompanyPartnerships(cmp *Company, tx *gorm.DB) []*Partnership {
	partnerships := make([]*Partnership, 0)

	tx.Preload("To").Preload("From").Where("`from_id` = ? or `to_id` = ?", cmp.ID, cmp.ID).Find(&partnerships)

	return partnerships
}

func Companies(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	companies := make([]*Company, 0)
	if err := tx.Order("share_capital desc").Find(&companies).Error; err != nil {
		panic(err)
	}

	for _, cmp := range companies {
		UpdateCompanyIncome(cmp, tx)
	}

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
	cmp.ActionPoints = opt.CompanyActionPoints + opt.InitialShares

	if err := tx.Create(cmp).Error; err != nil {
		panic(err)
	}

	if err := tx.Save(header.CurrentPlayer).Error; err != nil {
		panic(err)
	}

	for i := 0; i < opt.InitialShares; i++ {
		if err := tx.Create(&Share{CompanyID: cmp.ID, OwnerID: header.CurrentPlayer.ID}).Error; err != nil {
			panic(err)
		}
	}

	if err := tx.Where("`owner_id` = 0 and `yield` = 1").Find(&freenodes).Error; err != nil {
		panic(err)
	}

	if len(freenodes) != 0 {
		freeneighbours := make(map[*Node]int)
		maxfreeneighbours := 0
		nodepool := make([]*Node, 0, len(freenodes))

		for _, n := range freenodes {
			freeneighb := 0
			if err := tx.Model(&Node{}).Where("`x` >= ? and `x` <= ? and `y` >= ? and `y` <= ? and `owner_id` = 0", n.X-2, n.X+2, n.Y-2, n.Y+2).Count(&freeneighb).Error; err != nil {
				panic(err)
			}

			freeneighbours[n] = freeneighb

			if freeneighb > maxfreeneighbours {
				maxfreeneighbours = freeneighb
			}
		}

		for n, neighb := range freeneighbours {
			if neighb == maxfreeneighbours {
				nodepool = append(nodepool, n)
			}
		}

		node := nodepool[rand.Intn(len(nodepool))]

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
	Redirect(w, r, "gamehome")
}

func PromoteCEO(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	cmp := &Company{}
	myshares := 0
	ceoshares := 0
	sh := &Share{}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if err := tx.Where(params.ID).First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		session.AddFlash("Societa' inesistente!", "error_")
		goto out
	}

	sh.CompanyID = cmp.ID
	sh.OwnerID = header.CurrentPlayer.ID

	if err := tx.Model(sh).Where(sh).Count(&myshares).Error; err != nil {
		panic(err)
	}

	sh.OwnerID = cmp.CEOID

	if err := tx.Model(sh).Where(sh).Count(&ceoshares).Error; err != nil {
		panic(err)
	}

	if myshares > ceoshares {
		cmp.CEOID = header.CurrentPlayer.ID
	} else {
		session.AddFlash("Azioni insufficienti!", "error_")
		goto out
	}

	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Sei diventato amministratore!", "error_")

out:
	session.Save(r, w)

	url, err := router.Get("company").URL("id", fmt.Sprint(params.ID))
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}
