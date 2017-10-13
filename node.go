package main

import (
	"errors"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	. "github.com/vincenscotti/impero/model"
	"math"
	"net/http"
	"strconv"
)

func GetCostsByYield(yield int, opt *Options) (BuyCost int, InvestCost int, NewYield int) {
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

func BuyNode(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	opt := GetOptions(r)
	session := GetSession(r)

	blerr := BLError{}

	params := struct {
		ID uint
		X  int
		Y  int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if target, err := router.Get("map").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	node := &Node{}
	cost := 0
	cmp := &Company{}
	isnodeadjacent := false
	adjacentnodes := make([]*Node, 0, 8)
	adjacentrentals := 0
	adjacentx := []int{params.X - 1, params.X, params.X + 1}
	adjacenty := []int{params.Y - 1, params.Y, params.Y + 1}

	if err := tx.Where(params.ID).First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		blerr.Message = "Societa' inesistente!"
		panic(blerr)
	}

	if err := tx.Where("`x` = ? and `y` = ?", params.X, params.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if node.ID == 0 {
		blerr.Message = "Cella inesistente!"
		panic(blerr)
	}

	if cmp.CEOID != header.CurrentPlayer.ID {
		blerr.Message = "Permessi insufficienti!"
		panic(blerr)
	}

	if cmp.ActionPoints < 1 {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	if node.ID == 0 {
		blerr.Message = "Cella inesistente!"
		panic(blerr)
	}

	if node.OwnerID == cmp.ID {
		blerr.Message = "Cella gia' posseduta!"
		panic(blerr)
	}

	cost, _, _ = GetCostsByYield(node.Yield, opt)
	if cmp.ShareCapital < cost {
		blerr.Message = "Capitale insufficiente!"
		panic(blerr)
	}

	// select all adjacent nodes
	if err := tx.Model(&Node{}).Where("`x` in (?) and `y` in (?)", adjacentx, adjacenty).Find(&adjacentnodes).Error; err != nil {
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
			if err := tx.Model(&Rental{}).Where("`node_id` = ? and `tenant_id` = ?", n.ID, cmp.ID).Count(&adjacentrentals).Error; err != nil {
				panic(err)
			}

			if adjacentrentals > 0 {
				isnodeadjacent = true
				break
			}
		}
	}

	if !isnodeadjacent {
		blerr.Message = "Cella non adiacente!"
		panic(blerr)
	}

	if node.OwnerID != 0 {
		r := &Rental{}
		r.NodeID = node.ID
		r.TenantID = cmp.ID

		if err := tx.Where(r).Find(r).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if node.OwnerID == cmp.ID || r.ID != 0 {
			blerr.Message = "Cella gia' acquistata!"
			panic(blerr)
		} else {
			if err := tx.Create(r).Error; err != nil {
				panic(err)
			}
		}
	} else {
		node.OwnerID = cmp.ID

		if err := tx.Save(node).Error; err != nil {
			panic(err)
		}
	}

	cmp.ActionPoints -= 1
	cmp.ShareCapital -= cost

	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Cella acquistata!", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}

func InvestNode(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)
	opt := GetOptions(r)

	blerr := BLError{}

	params := struct {
		ID uint
		X  int
		Y  int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	if target, err := router.Get("map").URL(); err != nil {
		panic(err)
	} else {
		blerr.Redirect = target
	}

	node := &Node{}
	cmp := &Company{}
	cost := 0
	newyield := 0

	if err := tx.Where(params.ID).First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		blerr.Message = "Societa' inesistente!"
		panic(blerr)
	}

	if err := tx.Where("`x` = ? and `y` = ?", params.X, params.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.CEOID != header.CurrentPlayer.ID {
		blerr.Message = "Permessi insufficienti!"
		panic(blerr)
	}

	if cmp.ActionPoints < 1 {
		blerr.Message = "Punti operazione insufficienti!"
		panic(blerr)
	}

	if node.ID == 0 {
		blerr.Message = "Cella inesistente!"
		panic(blerr)
	}

	if node.OwnerID != cmp.ID {
		blerr.Message = "Cella non posseduta!"
		panic(blerr)
	}

	_, cost, newyield = GetCostsByYield(node.Yield, opt)

	if cmp.ShareCapital < cost {
		blerr.Message = "Capitale insufficiente!"
		panic(blerr)
	}

	cmp.ActionPoints -= 1
	cmp.ShareCapital -= cost
	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	node.Yield = newyield
	if err := tx.Save(node).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Cella migliorata!", "success_")

	RedirectToURL(w, r, blerr.Redirect)
}

func GetCosts(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	opt := GetOptions(r)

	ret := struct {
		BuyCost    int
		InvestCost int
	}{}

	params := mux.Vars(r)

	x, err := strconv.Atoi(params["x"])
	if err != nil {
		panic(err)
	}

	y, err := strconv.Atoi(params["y"])
	if err != nil {
		panic(err)
	}

	node := &Node{}

	if err := tx.Where("`x` = ? and `y` = ?", x, y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if node.ID != 0 {
		ret.BuyCost, ret.InvestCost, _ = GetCostsByYield(node.Yield, opt)
	}

	RenderJSON(w, r, ret)
}
