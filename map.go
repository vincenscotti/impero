package main

import (
	"github.com/gorilla/context"
	. "impero/model"
	"impero/templates"
	"net/http"
)

var ColourValues = []int32{
	0xFF0000, 0x00FF00, 0x0000FF, 0xFFFF00, 0xFF00FF, 0x00FFFF, /*0x000000,*/
	0x800000, 0x008000, 0x000080, 0x808000, 0x800080, 0x008080, 0x808080,
	0xC00000, 0x00C000, 0x0000C0, 0xC0C000, 0xC000C0, 0x00C0C0, 0xC0C0C0,
	0x400000, 0x004000, 0x000040, 0x404000, 0x400040, 0x004040, 0x404040,
	0x200000, 0x002000, 0x000020, 0x202000, 0x200020, 0x002020, 0x202020,
	0x600000, 0x006000, 0x000060, 0x606000, 0x600060, 0x006060, 0x606060,
	0xA00000, 0x00A000, 0x0000A0, 0xA0A000, 0xA000A0, 0x00A0A0, 0xA0A0A0,
	0xE00000, 0x00E000, 0x0000E0, 0xE0E000, 0xE000E0, 0x00E0E0, 0xE0E0E0,
}

func GetMap(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	//cmp := &Company{}
	s := struct {
		Minx int
		Miny int
		Maxx int
		Maxy int
	}{0, 0, 0, 0}

	nodes := make([]*Node, 0)
	rentals := make([]*Rental, 0)
	companiesbyname := make(map[string]*Company)

	tx.Preload("Owner").Find(&nodes)
	tx.Preload("Node").Preload("Tenant").Find(&rentals)

	colorn := 0
	for _, n := range nodes {
		if n.Owner.ID != 0 {
			_, ok := companiesbyname[n.Owner.Name]

			if !ok {
				companiesbyname[n.Owner.Name] = &n.Owner
				n.Owner.Color = ColourValues[colorn%len(ColourValues)]
				colorn += 1
			}
		}
	}

	tx.Raw("select min(x) as minx, min(y) as miny, max(x) as maxx, max(y) as maxy from nodes").Scan(&s)

	page := MapData{HeaderData: header, Nodes: nodes, Rentals: rentals, CompaniesByName: companiesbyname, XMin: s.Minx, YMin: s.Miny, XMax: s.Maxx, YMax: s.Maxy}

	renderHTML(w, 200, templates.MapPage(&page))
}
