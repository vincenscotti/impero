package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/vincenscotti/impero/engine"
	. "github.com/vincenscotti/impero/model"
	"net/http"
	"strconv"
)

func BuyNode(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	cmp := &Company{}
	cmp.ID = params.ID

	if err := tx.BuyNode(header.CurrentPlayer, cmp, engine.Coord{X: params.X, Y: params.Y}); err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		tx.Commit()

		session.AddFlash("Cella acquistata!", "success_")
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func InvestNode(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	session := GetSession(r)

	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	cmp := &Company{}
	cmp.ID = params.ID

	if err := tx.InvestNode(header.CurrentPlayer, cmp, engine.Coord{X: params.X, Y: params.Y}); err != nil {
		session.AddFlash(err.Error(), "error_")
	} else {
		tx.Commit()

		session.AddFlash("Cella migliorata!", "success_")
	}

	RedirectToURL(w, r, blerr.Redirect)
}

func GetCosts(w http.ResponseWriter, r *http.Request) {
	tx := gameEngine.OpenSession()
	defer tx.Close()

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

	_, ret.BuyCost, ret.InvestCost = tx.GetNodeCosts(engine.Coord{X: x, Y: y})

	RenderJSON(w, r, ret)
}
