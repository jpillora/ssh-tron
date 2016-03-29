package tron

import "sort"

const (
	sidebarWidth       = 14
	sidebarEntryHeight = 4
)

//
type scoreboard struct {
	g                *Game
	changed          bool
	allPlayersSorted []*Player
}

//
func (s *scoreboard) compute() {
	// pull in player list and sort
	sorted := make([]*Player, len(s.g.allPlayers))
	i := 0
	for _, p := range s.g.allPlayers {
		sorted[i] = p
		i++
	}
	sort.Sort(byScore(sorted))
	if s.allPlayersSorted == nil {
		s.changed = true
	}
	if len(sorted) > 0 {
		//place a rank on all players
		last := sorted[0]
		if last.index != 0 {
			s.changed = true
		}
		last.rank = 1
		for i = 1; i < len(sorted); i++ {
			p := sorted[i]
			if p.Kills == last.Kills &&
				p.Deaths == last.Deaths {
				p.rank = last.rank
			} else {
				p.rank = last.rank + 1
			}
			if p.index != i {
				s.changed = true
			}
			p.index = i
		}
	}
	if s.g.bot.connected && s.changed {
		s.g.bot.scoreChange(sorted)
	}
	s.allPlayersSorted = sorted
}

// byScore implements the sort.Interface to sort the players by score.
// The scores are first influenced by kills, then deaths and then names.
type byScore []*Player

func (ps byScore) Len() int      { return len(ps) }
func (ps byScore) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
func (ps byScore) Less(i, j int) bool {
	if ps[i].Kills > ps[j].Kills {
		return true
	} else if ps[i].Kills < ps[j].Kills {
		return false
	} else if ps[i].Deaths < ps[j].Deaths {
		return true
	} else if ps[i].Deaths > ps[j].Deaths {
		return false
	}
	return ps[i].hash < ps[j].hash
}
