package tron

import (
	"sort"
	"strconv"
)

const (
	sidebarWidth       = 14
	sidebarEntryHeight = 4
)

type sidebar struct {
	g       *Game
	ps      []*Player
	height  int
	runes   [][]rune
	changed bool
}

func (sb *sidebar) render() {

	// copy all players into a score sorted list
	sb.ps = make([]*Player, len(sb.g.players))
	i := 0
	for _, p := range sb.g.players {
		sb.ps[i] = p
		i++
	}

	sort.Sort(byScore(sb.ps))

	changed := false

	for i, p := range sb.ps {
		row := i * sidebarEntryHeight
		// skip players who render past the bottom of the screen
		if row+sidebarEntryHeight-1 >= sb.height {
			break
		}
		// calculate player stats
		r0 := []rune(" #" + strconv.Itoa(i+1) + " " + p.name)
		r1 := []rune("  " + p.status())
		r2 := []rune("  " + strconv.Itoa(p.kills) + " K/D " + strconv.Itoa(p.deaths))
		// compare against last
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

// byScore implements the sort.Interface to sort the players by score.
// The scores are first influenced by kills, then deaths and then names.
type byScore []*Player

func (ps byScore) Len() int      { return len(ps) }
func (ps byScore) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
func (ps byScore) Less(i, j int) bool {
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
