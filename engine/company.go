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

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
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
	cmp.ActionPoints = es.opt.CompanyActionPoints + 1 // one initial shareholder
	cmp.PureIncomePercentage = es.opt.CompanyPureIncomePercentage
	cmp.Shareholders = []Shareholder{Shareholder{Player: *p, Shares: es.opt.InitialShares}}

	if err := es.tx.Create(cmp).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Save(p).Error; err != nil {
		panic(err)
	}

	if err := es.tx.Where("`owner_id` = 0 and `yield` = 100").Find(&freenodes).Error; err != nil {
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

func (es *EngineSession) GetCompany(id int) (err error, cmp *Company, pureincome, valuepershare int) {
	cmp = &Company{}

	if err := es.tx.Preload("CEO").Where(id).First(cmp).Error; err != nil {
		panic(err)
	}

	es.tx.Model(cmp).Preload("Player").Related(&cmp.Shareholders)

	err, cmp.Income = es.GetCompanyIncome(cmp, false)
	if err != nil {
		return
	}

	err, pureincome, valuepershare = es.GetCompanyFinancials(cmp, false)
	if err != nil {
		return
	}

	return
}

func (es *EngineSession) GetCompanyPartnerships(cmp *Company) (err error, partnerships []*Partnership) {
	partnerships = make([]*Partnership, 0)

	es.tx.Preload("To").Preload("From").Where("`from_id` = ? or `to_id` = ?", cmp.ID, cmp.ID).Find(&partnerships)

	return nil, partnerships
}

func (es *EngineSession) GetCompanies() (err error, companies []*Company) {
	companies = make([]*Company, 0)
	if err := es.tx.Order("share_capital desc").Find(&companies).Error; err != nil {
		panic(err)
	}

	for _, cmp := range companies {
		_, cmp.Income = es.GetCompanyIncome(cmp, false)
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
	newceosh := &Shareholder{}
	oldceosh := &Shareholder{}

	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
	}

	if err := es.tx.First(cmp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("Societa' inesistente!")
		} else {
			panic(err)
		}
	}

	newceosh.CompanyID = cmp.ID
	newceosh.PlayerID = newceo.ID

	if err := es.tx.Where(newceosh).Find(&newceosh).Error; err != nil {
		panic(err)
	}

	newceoshares = newceosh.Shares

	oldceosh.CompanyID = cmp.ID
	oldceosh.PlayerID = cmp.CEOID

	if err := es.tx.Where(oldceosh).Find(&oldceosh).Error; err != nil {
		panic(err)
	}

	oldceoshares = oldceosh.Shares

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

func (es *EngineSession) GetCompanyIncome(cmp *Company, effective bool) (err error, income int) {
	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)

	if err := es.tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
		panic(err)
	}

	for _, n := range nodes {
		yield := n.Yield
		if effective {
			yield = effectiveYield(n)
		}

		income += yield

		if err := es.tx.Where("`node_id` = ?", n.ID).Find(&rentals).Error; err != nil {
			panic(err)
		}

		for _, _ = range rentals {
			income += yield / 2
		}
	}

	if err := es.tx.Preload("Node").Where("`tenant_id` = ?", cmp.ID).Find(&rentals).Error; err != nil {
		panic(err)
	}

	for _, r := range rentals {
		yield := r.Node.Yield
		if effective {
			yield = effectiveYield(&r.Node)
		}
		income += yield / 2
	}

	return
}

func (es *EngineSession) GetCompanyFinancials(cmp *Company, effective bool) (err error, pureIncome, valuePerShare int) {
	if cmp.Income == 0 {
		err, cmp.Income = es.GetCompanyIncome(cmp, effective)

		if err != nil {
			return
		}
	}

	if len(cmp.Shareholders) == 0 {
		es.tx.Model(cmp).Related(&cmp.Shareholders)
	}

	cmpshares := 0

	for _, sh := range cmp.Shareholders {
		cmpshares += sh.Shares
	}

	pureIncome = int(math.Floor(float64(cmp.Income) * (float64(cmp.PureIncomePercentage) / 100.0)))
	valuePerShare = (cmp.Income - pureIncome) / cmpshares

	return
}

func (es *EngineSession) ModifyCompanyPureIncome(p *Player, cmp *Company, increase bool) error {
	if es.timestamp.Before(es.opt.GameStart) {
		return errors.New("Il gioco non e' iniziato!")
	}

	if es.opt.Turn > es.opt.EndGame {
		return errors.New("Il gioco e' terminato!")
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
