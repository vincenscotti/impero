package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/jinzhu/gorm"
	. "impero/model"
	"net/http"
)

func BuyNode(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	opt := GetOptions(r)
	session := GetSession(r)

	params := struct {
		ID uint
		X  int
		Y  int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	node := &Node{}
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
		session.AddFlash("Societa' inesistente!", "error_")
		goto out
	}

	if err := tx.Where("`x` = ? and `y` = ?", params.X, params.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if node.ID == 0 {
		session.AddFlash("Cella inesistente!", "error_")
		goto out
	}

	if cmp.CEOID != header.CurrentPlayer.ID {
		session.AddFlash("Permessi insufficienti!", "error_")
		goto out
	}

	if cmp.ActionPoints < 1 {
		session.AddFlash("Punti operazione insufficienti!", "error_")
		goto out
	}

	if node.ID == 0 {
		session.AddFlash("Cella inesistente!", "error_")
		goto out
	}

	if node.OwnerID == cmp.ID {
		session.AddFlash("Cella gia' posseduta!", "error_")
		goto out
	}

	if cmp.ShareCapital < node.Yield*opt.CostPerYield {
		session.AddFlash("Capitale insufficiente!", "error_")
		goto out
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
		session.AddFlash("Cella non adiacente!", "error_")
		goto out
	}

	if node.OwnerID != 0 {
		r := &Rental{}
		r.NodeID = node.ID
		r.TenantID = cmp.ID

		// 'record not found' allowed
		if err := tx.Where(r).Find(r).Error; err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}

		if node.OwnerID == cmp.ID || r.ID != 0 {
			session.AddFlash("Cella gia' acquistata!", "error_")

			goto out
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
	cmp.ShareCapital -= node.Yield * 2
	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Cella acquistata!", "success_")

out:
	session.Save(r, w)

	if ref := r.Referer(); ref != "" {
		http.Redirect(w, r, ref, http.StatusFound)
	} else {
		url, err := router.Get("company").URL("id", fmt.Sprint(params.ID))
		if err != nil {
			panic(err)
		}

		http.Redirect(w, r, url.Path, http.StatusFound)
	}
}

func InvestNode(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	params := struct {
		ID uint
		X  int
		Y  int
	}{}

	if err := binder.Bind(&params, r); err != nil {
		panic(err)
	}

	node := &Node{}
	cmp := &Company{}

	if err := tx.Where(params.ID).First(cmp).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.ID == 0 {
		session.AddFlash("Societa' inesistente!", "error_")
		goto out
	}

	if err := tx.Where("`x` = ? and `y` = ?", params.X, params.Y).First(node).Error; err != nil && err != gorm.ErrRecordNotFound {
		panic(err)
	}

	if cmp.CEOID != header.CurrentPlayer.ID {
		session.AddFlash("Permessi insufficienti!", "error_")
		goto out
	}

	if cmp.ActionPoints < 1 {
		session.AddFlash("Punti operazione insufficienti!", "error_")
		goto out
	}

	if node.ID == 0 {
		session.AddFlash("Cella inesistente!", "error_")
		goto out
	}

	if node.OwnerID != cmp.ID {
		session.AddFlash("Cella non posseduta!", "error_")
		goto out
	}

	if cmp.ShareCapital < 2 {
		session.AddFlash("Capitale insufficiente!", "error_")
		goto out
	}

	cmp.ActionPoints -= 1
	cmp.ShareCapital -= 2
	if err := tx.Save(cmp).Error; err != nil {
		panic(err)
	}

	node.Yield += 2
	if err := tx.Save(node).Error; err != nil {
		panic(err)
	}

	session.AddFlash("Cella migliorata!", "success_")

out:
	session.Save(r, w)

	if ref := r.Referer(); ref != "" {
		http.Redirect(w, r, ref, http.StatusFound)
	} else {
		url, err := router.Get("company").URL("id", fmt.Sprint(params.ID))
		if err != nil {
			panic(err)
		}

		http.Redirect(w, r, url.Path, http.StatusFound)
	}
}
