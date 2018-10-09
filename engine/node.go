package engine

import (
	"errors"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"math"
)

var NodeYields = []struct {
	Yield       int
	Prob        float64
	UpgradeCost int
}{
	{1, 0.22, 1},
	{3, 0.5, 2},
	{6, 0.15, 5},
	{12, 0.08, 13},
	{25, 0.04, 30},
	{50, 0.01, 0},
}

func (es *EngineSession) GetCostsByYield(yield int) (BuyCost int, InvestCost int, NewYield int) {
	_, opt := es.GetOptions()
	BuyCost, InvestCost, NewYield = -1, -1, -1

	yieldindex := 0
	newyieldindex := 0
	yieldfound := false

	for i, y := range NodeYields {
		if y.Yield == yield {
			yieldfound = true
			yieldindex = i
			break
		}
	}

	if !yieldfound {
		panic(errors.New("Invalid yield value"))
	}

	newyieldindex = yieldindex + 1

	if newyieldindex < len(NodeYields) {
		InvestCost = NodeYields[yieldindex].UpgradeCost
		NewYield = NodeYields[newyieldindex].Yield
	}

	BuyCost = int(math.Floor(float64(yield) * opt.CostPerYield))

	return
}

func (es *EngineSession) GetNodeCosts(coord Coord) (err error, buyCost, investCost int) {
	node := &Node{}

	if err := es.tx.Where("`x` = ? and `y` = ?", coord.X, coord.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if node.ID == 0 {
		err = errors.New("Cella non trovata!")

		return
	}

	buyCost, investCost, _ = es.GetCostsByYield(node.Yield)

	return
}

func (es *EngineSession) BuyNode(p *Player, cmp *Company, coord Coord) error {
	node := &Node{}
	cost := 0
	isnodeadjacent := false
	adjacentnodes := make([]*Node, 0, 8)
	adjacentrentals := 0
	adjacentx := []int{coord.X - 1, coord.X, coord.X + 1}
	adjacenty := []int{coord.Y - 1, coord.Y, coord.Y + 1}

	if err := es.tx.First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		return errors.New("Societa' inesistente!")
	}

	if err := es.tx.Where("`x` = ? and `y` = ?", coord.X, coord.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if node.ID == 0 {
		return errors.New("Cella inesistente!")
	}

	if cmp.CEOID != p.ID {
		return errors.New("Permessi insufficienti!")
	}

	if cmp.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	if node.ID == 0 {
		return errors.New("Cella inesistente!")
	}

	if node.OwnerID == cmp.ID {
		return errors.New("Cella gia' posseduta!")
	}

	cost, _, _ = es.GetCostsByYield(node.Yield)
	if cmp.ShareCapital < cost {
		return errors.New("Capitale insufficiente!")
	}

	// select all adjacent nodes
	if err := es.tx.Model(&Node{}).Where("`x` in (?) and `y` in (?)", adjacentx, adjacenty).Find(&adjacentnodes).Error; err != nil {
		panic(err)
	}

	for _, n := range adjacentnodes {
		if n.OwnerID == cmp.ID {
			isnodeadjacent = true
			break
		}
	}

	// do I own any?
	if !isnodeadjacent {
		// nop, search for rentals then

		for _, n := range adjacentnodes {
			if err := es.tx.Model(&Rental{}).Where("`node_id` = ? and `tenant_id` = ?", n.ID, cmp.ID).Count(&adjacentrentals).Error; err != nil {
				panic(err)
			}

			if adjacentrentals > 0 {
				isnodeadjacent = true
				break
			}
		}
	}

	if !isnodeadjacent {
		return errors.New("Cella non adiacente!")
	}

	if node.OwnerID != 0 {
		r := &Rental{}
		r.NodeID = node.ID
		r.TenantID = cmp.ID

		if err := es.tx.Where(r).Find(r).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if node.OwnerID == cmp.ID || r.ID != 0 {
			return errors.New("Cella gia' acquistata!")
		} else {
			if err := es.tx.Create(r).Error; err != nil {
				panic(err)
			}
		}
	} else {
		node.OwnerID = cmp.ID

		if err := es.tx.Save(node).Error; err != nil {
			panic(err)
		}
	}

	cmp.ActionPoints -= 1
	cmp.ShareCapital -= cost

	if err := es.tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	return nil
}

func (es *EngineSession) InvestNode(p *Player, cmp *Company, coord Coord) error {
	node := &Node{}
	cost := 0
	newyield := 0

	if err := es.tx.First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		return errors.New("Societa' inesistente!")
	}

	if err := es.tx.Where("`x` = ? and `y` = ?", coord.X, coord.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.CEOID != p.ID {
		return errors.New("Permessi insufficienti!")
	}

	if cmp.ActionPoints < 1 {
		return errors.New("Punti operazione insufficienti!")
	}

	if node.ID == 0 {
		return errors.New("Cella inesistente!")
	}

	if node.OwnerID != cmp.ID {
		return errors.New("Cella non posseduta!")
	}

	_, cost, newyield = es.GetCostsByYield(node.Yield)

	if cmp.ShareCapital < cost {
		return errors.New("Capitale insufficiente!")
	}

	cmp.ActionPoints -= 1
	cmp.ShareCapital -= cost
	if err := es.tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	node.Yield = newyield
	if err := es.tx.Save(node).Error; err != nil {
		panic(err)
	}

	return nil
}
