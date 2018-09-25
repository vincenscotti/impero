package engine

import (
	. "github.com/vincenscotti/impero/model"
	"math"
	"math/rand"
)

var nodeYields = []struct {
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
