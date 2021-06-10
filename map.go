package main

import (
	"net/http"

	"github.com/gorilla/context"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
)

func (s *httpBackend) GetMap(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	_, mapnodes, rentals, companiesbyname, p1, p2 := tx.GetMapInfo()

	_, shares := tx.GetSharesForPlayer(header.CurrentPlayer)
	mycompanies := make(map[uint]bool)
	for _, sh := range shares {
		mycompanies[sh.CompanyID] = true
	}

	page := MapData{HeaderData: header, Nodes: mapnodes, Rentals: rentals, CompaniesByName: companiesbyname, MyCompanies: mycompanies, XMin: p1.X, YMin: p1.Y, XMax: p2.X, YMax: p2.Y}

	RenderHTML(w, r, templates.MapPage(&page))
}
