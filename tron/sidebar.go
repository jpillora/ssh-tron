package tron

import (
	"sort"
	"strconv"
)

const sidebarWidth = 14
const sidebarEntryHeight = 4

type Sidebar struct {
	g       *Game
	ps      []*Player
	height  int
	runes   [][]rune
	changed bool
}

func NewSidebar(g *Game) *Sidebar {
	//sidebar height - top and bottom rows are borders
	h := g.h - 2
	sb := &Sidebar{g: g, height: h, runes: make([][]rune, h)}
	return sb
}

func (sb *Sidebar) render() {

	//copy all players into a score sorted list
	sb.ps = make([]*Player, len(sb.g.players))

	i := 0
	for _, p := range sb.g.players {
		sb.ps[i] = p
		i++
	}

	sort.Sort(ByScore(sb.ps))

	changed := false

	for i, p := range sb.ps {
		row := i * sidebarEntryHeight
		//skip players who render past the bottom of the screen
		if row+sidebarEntryHeight-1 >= sb.height {
			break
		}
		//calculate player stats
		r0 := []rune(" #" + strconv.Itoa(i+1) + " " + p.name)
		r1 := []rune("  " + p.status())
		r2 := []rune("  " + strconv.Itoa(p.kills) + " K/D " + strconv.Itoa(p.deaths))
		//compare against last
		if !compare(r0, sb.runes[row+0]) ||
			!compare(r1, sb.runes[row+1]) ||
			!compare(r2, sb.runes[row+2]) {
			sb.runes[row+0] = r0
			sb.runes[row+1] = r1
			sb.runes[row+2] = r2
			changed = true
		}
	}

	sb.changed = changed
}

func compare(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

//sort players by score (kills then deaths then name)
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
	return ps[i].id < ps[j].id
}
