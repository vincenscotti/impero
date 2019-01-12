package engine

import (
	. "github.com/vincenscotti/impero/model"
	"io/ioutil"
	"math"
	"math/rand"
)

var nodeYields = []struct {
	Yield       int
	Prob        float64
	UpgradeCost int
}{
	{100, 0.22, 100},
	{300, 0.50, 200},
	{600, 0.15, 500},
	{1200, 0.08, 1300},
	{2500, 0.04, 3000},
	{5000, 0.01, 0},
}

var ColourValues = []int32{
	0xFF0000, 0x00FF00, 0x0000FF, 0xFFFF00, 0xFF00FF, 0x00FFFF, /*0x000000*/
	0x800000, 0x008000 /*0x000080 0x808000,*/, 0x800080, 0x008080, /*0x808080,*/
	0xC00000, 0x00C000, 0x0000C0, 0xC0C000 /*0xC000C0*/, 0x00C0C0, 0xC0C0C0,
	/*0x400000*/ 0x004000 /*0x000040*/, 0x404000 /*0x400040*/, 0x004040, 0x404040,
	/*0x200000 0x002000, 0x000020, 0x202000, 0x200020, 0x002020, 0x202020*/
	0x600000, 0x006000 /*0x000060*/, 0x606000, 0x600060, 0x006060, 0x606060,
	0xA00000, 0x00A000, 0x0000A0, 0xA0A000, 0xA000A0, 0x00A0A0, 0xA0A0A0,
	0xE00000, 0x00E000 /*0x0000E0*/, 0xE0E000, 0xE000E0, 0x00E0E0, 0xE0E0E0,
}

func (es *EngineSession) GetMapInfo() (err error, mapnodes map[Coord]*Node, rentals []*Rental, companies map[string]*Company, p1, p2 Coord) {
	//cmp := &Company{}
	s := struct {
		Minx int
		Miny int
		Maxx int
		Maxy int
	}{0, 0, 0, 0}

	nodes := make([]*Node, 0)
	mapnodes = make(map[Coord]*Node)
	rentals = make([]*Rental, 0)
	companies = make(map[string]*Company)

	es.tx.Find(&nodes)
	es.tx.Preload("Node").Preload("Tenant").Find(&rentals)

	colorn := 0
	for _, n := range nodes {
		n.BuyCost, n.InvestCost, n.NewYield = es.GetCostsByYield(n.Yield)
		mapnodes[Coord{X: n.X, Y: n.Y}] = n

		// cannot Preload, cause of sqlite bug
		if n.OwnerID != 0 {
			if n.Owner.ID == 0 {
				es.tx.Where(n.OwnerID).Find(&n.Owner)
			}

			_, ok := companies[n.Owner.Name]

			if !ok {
				companies[n.Owner.Name] = &n.Owner
				n.Owner.Color = ColourValues[colorn%len(ColourValues)]
				colorn += 1
			}
		}
	}

	es.tx.Raw("select min(x) as minx, min(y) as miny, max(x) as maxx, max(y) as maxy from nodes").Scan(&s)

	p1.X = s.Minx
	p1.Y = s.Miny
	p2.X = s.Maxx
	p2.Y = s.Maxy

	return
}

func (es *EngineSession) ImportMap() (err error) {
	var sql []byte

	sql, err = ioutil.ReadFile("map.sql")

	if err != nil {
		return
	}

	return es.tx.Exec(string(sql)).Error
}

func (es *EngineSession) UpdateMapYields(x0, x1, y0, y1 int, generate bool) error {
	xsize := x1 - x0
	ysize := y1 - y0
	sortednodes := make([]*Node, 0, xsize*ysize)

	if generate {
		for i := x0; i < x1; i++ {
			for j := y0; j < y1; j++ {
				sortednodes = append(sortednodes, &Node{X: i, Y: j})
			}
		}
	} else {
		if err := es.tx.Where("`x` > ? and `x` < ? and `y` > ? and `y` < ?",
			x0, x1, y0, y1).Find(&sortednodes).Error; err != nil {
			panic(err)
		}
	}

	perm := rand.Perm(len(sortednodes))

	shufflednodes := make([]*Node, len(sortednodes))

	for i, p := range perm {
		shufflednodes[i] = sortednodes[p]
	}

	totalnodes := float64(len(shufflednodes))
	remainingnodes := totalnodes
	maxyield := 0

	for _, y := range nodeYields {
		yield := y.Yield
		prob := y.Prob

		if yield > maxyield {
			maxyield = yield
		}

		nodesperyield := int(math.Floor(totalnodes * prob))

		for nodesperyield > 0 {
			shufflednodes[0].Yield = yield
			if err := es.tx.Save(shufflednodes[0]).Error; err != nil {
				panic(err)
			}

			shufflednodes = shufflednodes[1:]
			nodesperyield -= 1
			remainingnodes -= 1
		}
	}

	for remainingnodes > 0 {
		shufflednodes[0].Yield = maxyield
		if err := es.tx.Save(shufflednodes[0]).Error; err != nil {
			panic(err)
		}

		shufflednodes = shufflednodes[1:]
		remainingnodes -= 1
	}

	return nil
}
