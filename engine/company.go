package engine

import (
	. "github.com/vincenscotti/impero/model"
	"math"
)

func (es *EngineSession) GetCompanyIncome(cmp *Company) (err error, income int) {
	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)

	if err := es.tx.Where("`owner_id` = ?", cmp.ID).Find(&nodes).Error; err != nil {
		panic(err)
	}

	for _, n := range nodes {
		income += n.Yield

		if err := es.tx.Where("`node_id` = ?", n.ID).Find(&rentals).Error; err != nil {
			panic(err)
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
	floatPureIncome := math.Floor(floatIncome * 0.3)
	floatValuePerShare := int(math.Ceil((floatIncome - floatPureIncome) / float64(cmpshares)))

	return nil, int(floatPureIncome), int(floatValuePerShare)
}
