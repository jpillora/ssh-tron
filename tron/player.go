//a player represents a live tcp
//connection from a client
package tron

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/jpillora/ansi"
	"golang.org/x/crypto/ssh"
)

var (
	filled = '⣿'
	top    = '⠛'
	bottom = '⣤'
	empty  = ' '
)

type Direction byte

const (
	dup Direction = iota + 65
	ddown
	dright
	dleft
)

var colours = map[ID][]byte{
	blank: ansi.Set(ansi.White),
	wall:  ansi.Set(ansi.White),
	ID(1): ansi.Set(ansi.Blue),
	ID(2): ansi.Set(ansi.Green),
	ID(3): ansi.Set(ansi.Magenta),
	ID(4): ansi.Set(ansi.Cyan),
	ID(5): ansi.Set(ansi.Yellow),
	ID(6): ansi.Set(ansi.Red),
}

type resize struct {
	width, height uint32
}

type Player struct {
	id                           ID // identification
	name, cname                  string
	x, y                         uint8     // position
	d, nextd                     Direction // direction
	w, h                         int       // terminal size
	screen                       [][]rune  // the player's view of the screen
	dead, ready, waiting, redraw bool      // player flags
	tdeath                       time.Time // time of death
	kills, deaths                int       // score
	playing                      chan bool // is playing signal
	g                            *Game
	resizes                      chan resize
	conn                         *ansi.Ansi
	logf                         func(format string, args ...interface{})
	once                         *sync.Once
}

func NewPlayer(id ID, name string, conn ssh.Channel) *Player {

	colouredName := fmt.Sprintf("%s%s%s", colours[id], name, ansi.Set(ansi.Reset))

	p := &Player{
		id:      id,
		name:    name,
		cname:   colouredName,
		dead:    true,
		ready:   false,
		playing: make(chan bool, 1),
		d:       dup,
		resizes: make(chan resize),
		conn:    ansi.Wrap(conn),
		logf:    log.New(os.Stdout, colouredName+" ", 0).Printf,
		once:    &sync.Once{},
	}
	return p
}

func (p *Player) resetScreen() {
	p.screen = make([][]rune, p.g.w)
	for w := 0; w < p.g.w; w++ {
		p.screen[w] = make([]rune, p.g.h)
		for h := 0; h < p.g.h; h++ {
			p.screen[w][h] = empty
		}
	}
	p.redraw = true
}

const (
	respawnAttempts  = 100
	respawnLookahead = 15
)

func (p *Player) respawn() {
	if !p.dead || !p.ready || p.waiting {
		return
	}

	for i := 0; i < respawnAttempts; i++ {
		// randomly spawn player
		p.x = uint8(rand.Intn(int(p.g.bw-2))) + 1
		p.y = uint8(rand.Intn(int(p.g.bh-2))) + 1
		p.d = Direction(uint8(rand.Intn(4) + 65))
		p.nextd = p.d

		// look ahead
		clear := true
		x, y := p.x, p.y
		for j := 0; j < respawnLookahead; j++ {
			switch p.d {
			case dup:
				y--
			case ddown:
				y++
			case dleft:
				x--
			case dright:
				x++
			}
			if p.g.board[x][y] != blank {
				clear = false
				break
			}
		}
		// when clear, mark player as alive
		if clear {
			p.dead = false
			break
		}
	}
}

func (p *Player) play() {
	p.logf("connected")

	p.conn.Set(ansi.Reset)
	p.conn.CursorHide()

	go p.resizeWatch()
	go p.recieveActions()

	// block until player disconnects
	<-p.playing
	p.logf("disconnected")
}

func (p *Player) teardown() {
	// guard teardown to execute only once per player
	p.once.Do(p.teardown_)
}

func (p *Player) teardown_() {
	p.conn.CursorShow()
	p.conn.EraseScreen()
	p.conn.Goto(1, 1)
	p.conn.Set(ansi.Reset)
	p.conn.Close()
	close(p.playing)
}

func (p *Player) status() string {
	if !p.ready {
		return "not ready"
	} else if p.dead && p.waiting {
		return fmt.Sprintf("dead %1.1f", (p.g.delay - time.Since(p.tdeath)).Seconds())
	} else if p.dead {
		return "ready"
	}
	return "playing"
}

func (p *Player) recieveActions() {
	buff := make([]byte, 0xffff)
	for {
		n, err := p.conn.Read(buff)
		if err != nil {
			break
		}
		b := buff[:n]
		if b[0] == 3 {
			break
		}
		// ignore actions until ready
		if !p.ready {
			continue
		}
		// parse up,down,left,right
		d := byte(p.d)
		if len(b) == 3 && b[0] == ansi.Esc && b[1] == 91 &&
			b[2] >= byte(dup) && b[2] <= byte(dleft) &&
			// while preventing player from moving into itself (odd<->even)
			((d%2 == 0 && d-1 != b[2]) || ((d+1)%2 == 0 && d+1 != b[2])) {
			p.nextd = Direction(b[2])
			continue
		}
		// respawn!
		if b[0] == 13 {
			p.respawn()
			continue
		}
		// p.logf("sent action %+v", b)
	}
	p.teardown()
}

var resizeTmpl = string(ansi.Goto(2, 5)) +
	string(ansi.Set(ansi.White)) +
	"Please resize your terminal to %dx%d (+%dx+%d)"

func (p *Player) resizeWatch() {

	for r := range p.resizes {
		p.w = int(r.width)
		p.h = int(r.height)
		// fits?
		if p.w >= p.g.w && p.h >= p.g.h {
			p.conn.EraseScreen()
			p.resetScreen()
			// send updates!
			p.ready = true
		} else {
			// doesnt fit
			p.conn.EraseScreen()
			p.conn.Write([]byte(fmt.Sprintf(resizeTmpl, p.g.w, p.g.h,
				int(math.Max(float64(p.g.w-p.w), 0)),
				int(math.Max(float64(p.g.h-p.h), 0)))))
			p.screen = nil
			p.ready = false
		}
	}
}

// every tick, based on player screen size - calculate, store and send screen deltas.
func (p *Player) update() {

	if !p.ready {
		return
	}

	gb := p.g.board

	// center board with offset width and height
	ow := (p.w - p.g.w) / 2
	oh := (p.h - p.g.h) / 2

	// store the last rendered for network optimisation
	var lastw, lasth uint16
	var r rune
	var c, lastc []byte

	// screen loop
	var u []byte
	for h := 0; h < p.g.h; h++ {
		for tw := 0; tw < p.g.w; tw++ {
			// rune and color at terminal w x h
			r = empty
			c = colours[blank]

			sidebar := false

			if tw < sidebarWidth {
				// calculate rune from sidebar
				if tw == 0 {
					r = filled
				} else if h == 0 {
					r = top
				} else if h == p.g.h-1 {
					r = bottom
				} else {

					rs := p.g.sidebar.runes
					if h-1 < len(rs) && tw-1 < len(rs[h-1]) {
						i := (h - 1) / sidebarEntryHeight
						if i < len(p.g.sidebar.ps) {
							p := p.g.sidebar.ps[i]
							c = colours[p.id]
							r = rs[h-1][tw-1]
							sidebar = true
						}
					}
				}

			} else {
				// calculate rune from game
				gw := tw - sidebarWidth
				h1 := h * 2
				h2 := h1 + 1
				// choose rune
				if gb[gw][h1] != blank && gb[gw][h2] != blank {
					r = filled
				} else if gb[gw][h1] != blank {
					r = top
				} else if gb[gw][h2] != blank {
					r = bottom
				}
				// choose color
				if gb[gw][h2] == blank {
					c = colours[gb[gw][h1]]
				} else {
					c = colours[gb[gw][h2]]
				}
			}

			cacheOverride := p.g.sidebar.changed && sidebar

			// player board is different? draw it
			if p.screen[tw][h] != r || cacheOverride {

				// p.logf("rune differs '%s' %dx%d (%v)", string(r), tw, h, cacheOverride)

				// skip if we only moved one space right
				nexth := uint16(h + 1 + oh)
				nextw := uint16(tw + 1 + ow)
				if nexth != lasth || nextw != lastw+1 {
					u = append(u, ansi.Goto(nexth, nextw)...)
					lasth = nexth
					lastw = nextw
				}
				// skip if we didnt change color
				if c != nil && bytes.Compare(c, lastc) != 0 {
					u = append(u, c...)
					lastc = c
				}

				// write rune
				u = append(u, []byte(string(r))...)
				// cache
				p.screen[tw][h] = r
			}
		}
	}

	if len(u) == 0 {
		return
	}

	// p.logf("send %d", len(u))

	p.conn.Write(u)
}
