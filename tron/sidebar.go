package tron

import (
	"sort"
	"strconv"
)

const sidebarWidth = 16
const sidebarEntryHeight = 3

type Sidebar struct {
	g       *Game
	ps      []*Player
	runes   [][]rune
	changed bool
}

func NewSidebar(g *Game) *Sidebar {
	sb := &Sidebar{g: g}
	return sb
}

func (sb *Sidebar) render() {

	sb.ps = make([]*Player, len(sb.g.players))

	i := 0
	for _, p := range sb.g.players {
		sb.ps[i] = p
		i++
	}

	sort.Sort(ByScore(sb.ps))

	h := sb.g.h - 2
	sb.runes = make([][]rune, h)

	for i, p := range sb.ps {
		row := i * sidebarEntryHeight
		if row+sidebarEntryHeight-1 >= h {
			break
		}
		sb.runes[row+0] = []rune(" #" + strconv.Itoa(i+1) + " " + p.name)
		sb.runes[row+1] = []rune("  " + strconv.Itoa(p.kills) + " K/D " + strconv.Itoa(p.deaths))
	}

	sb.changed = true
}

//sort players by score (kills then deaths)
type ByScore []*Player

func (ps ByScore) Len() int      { return len(ps) }
func (ps ByScore) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
func (ps ByScore) Less(i, j int) bool {
	if ps[i].kills > ps[j].kills {
		return true
	} else if ps[i].kills < ps[j].kills {
		return false
	} else if ps[i].deaths < ps[j].deaths {
		return true
	} else if ps[i].deaths > ps[j].deaths {
		return false
	}
	return ps[i].id < ps[i].id
}
