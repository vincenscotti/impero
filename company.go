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
	"time"
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

	blerr := BLError{}

	if target, err := router.Get("gamehome").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	cmp := &Company{}
	freenodes := make([]*Node, 0)
	cnt := 0

	if err := binder.Bind(cmp, r); err != nil {
		panic(err)
	}

	if cmp.Name == "" {
		blerr.Message = "Il nome non puo' essere vuoto!"
		panic(blerr)
	}

	if cmp.ShareCapital < 1 {
		blerr.Message = "Il budget deve essere almeno 1!"
		panic(blerr)
	}

	if cmp.ShareCapital > header.CurrentPlayer.Budget {
		blerr.Message = "Budget insufficiente!"
		panic(blerr)
	}

	if header.CurrentPlayer.ActionPoints < opt.NewCompanyCost {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	if err := tx.Model(cmp).Where(&Company{Name: cmp.Name}).Count(&cnt).Error; err != nil {
		panic(err)
	}

	if cnt != 0 {
		blerr.Message = "Societa' gia' esistente!"
		panic(blerr)
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
		blerr.Message = "Nessuna cella disponibile!"
		panic(blerr)
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func PromoteCEO(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	blerr := BLError{}

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

	if target, err := router.Get("company").URL("id", fmt.Sprint(params.ID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err := tx.Where(params.ID).First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		blerr.Message = "Societa' inesistente!"
		panic(blerr)
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
		blerr.Message = "Azioni insufficienti!"
		panic(blerr)
	}

	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Sei diventato amministratore!", "error_")

	RedirectToURL(w, r, blerr.Redirect)
}

func ProposePartnership(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)
	now := GetTime(r)
	opt := GetOptions(r)
	header := context.Get(r, "header").(*HeaderData)

	blerr := BLError{}

	p := &Partnership{}
	from := &Company{}
	to := &Company{}
	count := 0

	if err := binder.Bind(p, r); err != nil {
		panic(err)
	}

	if target, err := router.Get("company").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if err := tx.Where(p.FromID).First(from).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if err := tx.Where(p.ToID).First(to).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	subject := "Proposta di partnership"
	content := "Hai ricevuto una proposta di partnership tra " + from.Name + " e " + to.Name
	report := &Report{PlayerID: to.CEOID, Date: now, Subject: subject, Content: content}

	if from.ID == 0 || to.ID == 0 {
		blerr.Message = "Societa' inesistente!"
		panic(blerr)
	}

	if from.CEOID != header.CurrentPlayer.ID {
		blerr.Message = "Non sei il CEO!"
		panic(blerr)
	}

	if from.CEOID == to.CEOID {
		blerr.Message = "Sei il CEO di entrambe le societa'!"
		panic(blerr)
	}

	if from.ActionPoints < 1 {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	if err := tx.Table("partnerships").Where("((`from_id` = ? and `to_id` = ?) or (`from_id` = ? and `to_id` = ?)) and `deleted_at` is null", from.ID, to.ID, to.ID, from.ID).Count(&count).Error; err != nil {
		panic(err)
	}

	if count > 0 {
		blerr.Message = "Partnership gia' esistente!"
		panic(blerr)
	}

	from.ActionPoints -= 1
	if err := tx.Save(from).Error; err != nil {
		panic(err)
	}

	if err := tx.Create(&Partnership{FromID: from.ID, ToID: to.ID,
		ProposalExpiration: now.Add(time.Duration(opt.TurnDuration) * time.Minute)}).Error; err != nil {
		panic(err)
	}

	if err := tx.Create(report).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Proposa inviata!", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}

func ConfirmPartnership(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)
	now := GetTime(r)
	header := context.Get(r, "header").(*HeaderData)

	blerr := BLError{}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	p := &Partnership{}

	if err := tx.Preload("To").Preload("From").Where(params.ID).First(p).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if target, err := router.Get("company").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if p.To.CEOID != header.CurrentPlayer.ID {
		blerr.Message = "Non sei il CEO!"
		panic(blerr)
	}

	if p.To.ActionPoints < 1 {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	p.To.ActionPoints -= 1
	if err := tx.Save(&p.To).Error; err != nil {
		panic(err)
	}

	p.ProposalExpiration = time.Time{}
	if err := tx.Save(p).Error; err != nil {
		panic(err)
	}

	subject := "Proposta di partnership confermata"
	content := "La proposta di partnership tra " + p.From.Name + " e " + p.To.Name + " e' stata confermata"
	report := &Report{PlayerID: p.From.CEOID, Date: now, Subject: subject, Content: content}

	if err := tx.Create(report).Error; err != nil {
		panic(err)
	}

	report.ID = 0
	report.PlayerID = p.To.CEOID

	if err := tx.Create(report).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Partnership confermata!", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}

func DeletePartnership(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	session := GetSession(r)
	now := GetTime(r)
	header := context.Get(r, "header").(*HeaderData)

	blerr := BLError{}

	params := struct {
		ID uint
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	p := &Partnership{}
	var cmp *Company

	if err := tx.Preload("To").Preload("From").Where(params.ID).First(p).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if target, err := router.Get("company").URL("id", fmt.Sprint(p.ToID)); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	if p.To.CEOID == header.CurrentPlayer.ID {
		cmp = &p.To
	} else if p.From.CEOID == header.CurrentPlayer.ID {
		cmp = &p.From
	} else {
		blerr.Message = "Non sei il CEO!"
		panic(blerr)
	}

	if cmp.ActionPoints < 1 {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	cmp.ActionPoints -= 1
	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	if err := tx.Delete(p).Error; err != nil {
		panic(err)
	}

	subject := "Partnership cancellata"
	content := "La partnership tra " + p.From.Name + " e " + p.To.Name + " e' stata cancellata"
	report := &Report{PlayerID: p.From.CEOID, Date: now, Subject: subject, Content: content}

	if err := tx.Create(report).Error; err != nil {
		panic(err)
	}

	report.ID = 0
	report.PlayerID = p.To.CEOID

	if err := tx.Create(report).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Partnership cancellata!", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}
