package model

type PlayersSortableByVP []*Player

func (ps PlayersSortableByVP) Len() int {
	return len([]*Player(ps))
}

func (ps PlayersSortableByVP) Less(i, j int) bool {
	p := []*Player(ps)
	return p[i].VP > p[j].VP
}

func (ps PlayersSortableByVP) Swap(i, j int) {
	p := []*Player(ps)

	p[i], p[j] = p[j], p[i]
}

type CompaniesSortableByIncome []*Company

func (cs CompaniesSortableByIncome) Len() int {
	return len([]*Company(cs))
}

func (cs CompaniesSortableByIncome) Less(i, j int) bool {
	c := []*Company(cs)
	return c[i].Income > c[j].Income
}

func (cs CompaniesSortableByIncome) Swap(i, j int) {
	c := []*Company(cs)

	c[i], c[j] = c[j], c[i]
}
