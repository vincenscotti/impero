package engine

import (
	"errors"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"math"
	"math/rand"
)

func (es *EngineSession) NewCompany(p *Player, name string, capital int) error {
	cmp := &Company{}
	freenodes := make([]*Node, 0)
	cnt := 0

	cmp.Name = name
	cmp.ShareCapital = capital

	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if cmp.Name == "" {
		return errors.New("Il nome non puo' essere vuoto!")
	}

	if cmp.ShareCapital < 1 {
		return errors.New("Il budget deve essere almeno 1!")
	}

	if cmp.ShareCapital > p.Budget {
		return errors.New("Budget insufficiente!")
	}

	if p.ActionPoints < es.opt.NewCompanyCost {
		return errors.New("Punti operazione insufficienti!")
	}

	if err := es.tx.Model(cmp).Where(&Company{Name: cmp.Name}).Count(&cnt).Error; err != nil {
		panic(err)
	}

	if cnt != 0 {
		return errors.New("Societa' gia' esistente!")
	}

	p.Budget -= cmp.ShareCapital
	p.ActionPoints -= es.opt.NewCompanyCost
	cmp.CEO = *p
	cmp.ActionPoints = es.opt.CompanyActionPoints + es.opt.InitialShares
	cmp.PureIncomePercentage = es.opt.CompanyPureIncomePercentage

	if err := es.tx.Create(cmp).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	for i := 0; i < es.opt.InitialShares; i++ {
		if err := es.tx.Create(&Share{CompanyID: cmp.ID, OwnerID: p.ID}).Error; err != nil {
			panic(err)
		}
	}

	if err := es.tx.Where("`owner_id` = 0 and `yield` = 1").Find(&freenodes).Error; err != nil {
		panic(err)
	}

	if len(freenodes) != 0 {
		freeneighbours := make(map[*Node]int)
		maxfreeneighbours := 0
		nodepool := make([]*Node, 0, len(freenodes))

		for _, n := range freenodes {
			freeneighb := 0
			if err := es.tx.Model(&Node{}).Where("`x` >= ? and `x` <= ? and `y` >= ? and `y` <= ? and `owner_id` = 0", n.X-2, n.X+2, n.Y-2, n.Y+2).Count(&freeneighb).Error; err != nil {
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

		if err := es.tx.Save(node).Error; err != nil {
			panic(err)
		}
	} else {
		return errors.New("Nessuna cella disponibile!")
	}

	return nil
}

func (es *EngineSession) GetCompany(p *Player, id int) (err error, cmp *Company, shareholders []*ShareholdersPerCompany, pureincome, valuepershare int) {
	cmp = &Company{}
	shares := 0
	myshares := 0

	if err := es.tx.Preload("CEO").Where(id).First(cmp).Error; err != nil {
		panic(err)
	}
	if err := es.tx.Model(&Share{}).Where("`company_id` = ?", id).Where("`owner_id` != 0").Count(&shares).Error; err != nil {
		panic(err)
	}
	if err := es.tx.Model(&Share{}).Where("`company_id` = ?", id).Where("`owner_id` = ?", p.ID).Count(&myshares).Error; err != nil {
		panic(err)
	}

	es.updateCompanyIncome(cmp)

	err, pureincome, valuepershare = es.GetCompanyFinancials(cmp)
	if err != nil {
		return
	}

	shareholders = make([]*ShareholdersPerCompany, 0)

	if err := es.tx.Table("shares").Select("DISTINCT owner_id, count(owner_id) as shares").
		Where("`company_id` = ? and `owner_id` != 0", cmp.ID).
		Group("`owner_id`").Order("`owner_id` asc").
		Find(&shareholders).Error; err != nil {
		panic(err)
	}

	for _, sh := range shareholders {
		if err := es.tx.Table("players").Where(sh.OwnerID).Find(&sh.Owner).Error; err != nil {
			panic(err)
		}
	}

	return
}

func (es *EngineSession) GetCompanyPartnerships(cmp *Company) (err error, partnerships []*Partnership) {
	partnerships = make([]*Partnership, 0)

	es.tx.Preload("To").Preload("From").Where("`from_id` = ? or `to_id` = ?", cmp.ID, cmp.ID).Find(&partnerships)

	return nil, partnerships
}

func (es *EngineSession) updateCompanyIncome(cmp *Company) {
	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)

	if err := es.tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
		panic(err)
	}

	income := 0

	for _, n := range nodes {
		income += n.Yield

		if err := es.tx.Where("`node_id` = ?", n.ID).Find(&rentals).Error; err != nil {
			panic(err.Error())
		}

		for _, _ = range rentals {
			income += int(math.Ceil(float64(n.Yield) / 2.))
		}
	}

	if err := es.tx.Preload("Node").Where("`tenant_id` = ?", cmp.ID).Find(&rentals).Error; err != nil {
		panic(err)
	}

	for _, r := range rentals {
		income += r.Node.Yield / 2
	}

	cmp.Income = income
}

func (es *EngineSession) GetCompanies() (err error, companies []*Company) {
	companies = make([]*Company, 0)
	if err := es.tx.Order("share_capital desc").Find(&companies).Error; err != nil {
		panic(err)
	}

	for _, cmp := range companies {
		es.GetCompanyIncome(cmp)
	}

	return
}

func (es *EngineSession) GetOwnedCompanies(p *Player) (err error, companies []*Company) {
	companies = make([]*Company, 0)

	es.tx.Where("`ceo_id` = ?", p.ID).Find(&companies)

	return
}

func (es *EngineSession) PromoteCEO(cmp *Company, newceo *Player) error {
	newceoshares := 0
	oldceoshares := 0
	sh := &Share{}

	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if err := es.tx.First(cmp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("Societa' inesistente!")
		} else {
			panic(err)
		}
	}

	sh.CompanyID = cmp.ID
	sh.OwnerID = newceo.ID

	if err := es.tx.Model(sh).Where(sh).Count(&newceoshares).Error; err != nil {
		panic(err)
	}

	sh.OwnerID = cmp.CEOID

	if err := es.tx.Model(sh).Where(sh).Count(&oldceoshares).Error; err != nil {
		panic(err)
	}

	if newceoshares > oldceoshares {
		cmp.CEOID = newceo.ID
	} else {
		return errors.New("Azioni insufficienti!")
	}

	if err := es.tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	return nil
}

func (es *EngineSession) GetCompanyIncome(cmp *Company) (err error, income int) {
	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)

	if err := es.tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
		panic(err)
	}

	for _, n := range nodes {
		yield := effectiveYield(n)
		income += yield

		if err := es.tx.Where("`node_id` = ?", n.ID).Find(&rentals).Error; err != nil {
			panic(err)
		}

		for _, _ = range rentals {
			income += int(math.Ceil(float64(yield) / 2.))
		}
	}

	if err := es.tx.Preload("Node").Where("`tenant_id` = ?", cmp.ID).Find(&rentals).Error; err != nil {
		panic(err)
	}

	for _, r := range rentals {
		income += effectiveYield(&r.Node) / 2
	}

	return
}

func (es *EngineSession) GetCompanyFinancials(cmp *Company) (err error, pureIncome, valuePerShare int) {
	if cmp.Income == 0 {
		err, cmp.Income = es.GetCompanyIncome(cmp)

		if err != nil {
			return
		}
	}

	cmpshares := 0

	if err := es.tx.Table("shares").Where("`company_id` = ?", cmp.ID).Where("`owner_id` != 0").Count(&cmpshares).Error; err != nil {
		panic(err)
	}

	floatIncome := float64(cmp.Income)
	floatPureIncome := math.Floor(floatIncome * (float64(cmp.PureIncomePercentage) / 100.0))
	floatValuePerShare := int(math.Ceil((floatIncome - floatPureIncome) / float64(cmpshares)))

	return nil, int(floatPureIncome), int(floatValuePerShare)
}

func (es *EngineSession) ModifyCompanyPureIncome(p *Player, cmp *Company, increase bool) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if err := es.tx.First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		return errors.New("Societa' inesistente!")
	}

	if cmp.CEOID != p.ID {
		return errors.New("Permessi insufficienti!")
	}

	if cmp.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	if increase && cmp.PureIncomePercentage == 100 {
		return errors.New("Non puoi incrementare ulteriormente la percentuale!")
	} else if !increase && cmp.PureIncomePercentage == 0 {
		return errors.New("Non puoi decrementare ulteriormente la percentuale!")
	}

	cmp.ActionPoints -= 1
	if increase {
		cmp.PureIncomePercentage += 10
	} else {
		cmp.PureIncomePercentage -= 10
	}

	if err := es.tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	return nil
}
