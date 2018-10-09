package main

import (
	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"net/http"
)

func GetMap(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)

	tx := gameEngine.OpenSession()
	defer tx.Close()

	_, mapnodes, rentals, companiesbyname, p1, p2 := tx.GetMapInfo()

	page := MapData{HeaderData: header, Nodes: mapnodes, Rentals: rentals, CompaniesByName: companiesbyname, XMin: p1.X, YMin: p1.Y, XMax: p2.X, YMax: p2.Y}

	RenderHTML(w, r, templates.MapPage(&page))
}
