package tron

import (
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

const slotHeight = 4

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

func (d Direction) String() string {
	switch d {
	case dup:
		return "up"
	case ddown:
		return "down"
	case dleft:
		return "left"
	case dright:
		return "right"
	default:
		return fmt.Sprintf("%d", d)
	}
}

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

// A Player represents a live TCP connection from a client
type Player struct {
	id                   ID     // identification
	hash                 string //hash of public key
	SSHName, Name, cname string
	rank, index          int
	x, y                 uint8     // position
	d                    Direction // curr direction
	nextd                Direction // next direction
	w, h                 int       // terminal size
	screenRunes          [][]rune  // the player's view of the screen
	screenColors         [][]ID    // the player's view of the screen
	score                [slotHeight]string
	scoreDrawn, redraw   bool
	dead, ready, waiting bool
	tdeath               time.Time // time of death
	Kills, Deaths        int       // score
	playing              chan bool // is playing signal
	g                    *Game
	resizes              chan resize
	conn                 *ansi.Ansi
	logf                 func(format string, args ...interface{})
	once                 *sync.Once
}

// NewPlayer returns an initialized Player.
func NewPlayer(id ID, sshName, name, hash string, conn ssh.Channel) *Player {
	if hash == "" {
		hash = name //finally, hash fallsback to name
	}
	colouredName := fmt.Sprintf("%s%s%s", colours[id], name, ansi.Set(ansi.Reset))
	p := &Player{
		id:      id,
		hash:    hash,
		SSHName: sshName,
		Name:    name,
		cname:   colouredName,
		d:       dup,
		dead:    true,
		ready:   false,
		playing: make(chan bool, 1),
		resizes: make(chan resize),
		conn:    ansi.Wrap(conn),
		logf:    log.New(os.Stdout, colouredName+" ", 0).Printf,
		once:    &sync.Once{},
	}
	return p
}

func (p *Player) resetScreen() {
	p.screenRunes = make([][]rune, p.g.w)
	p.screenColors = make([][]ID, p.g.w)
	for w := 0; w < p.g.w; w++ {
		p.screenRunes[w] = make([]rune, p.g.h)
		p.screenColors[w] = make([]ID, p.g.h)
		for h := 0; h < p.g.h; h++ {
			p.screenRunes[w][h] = empty
			p.screenColors[w][h] = ID(255)
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
	p.once.Do(p.teardownMeta)
}

func (p *Player) teardownMeta() {
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
		return fmt.Sprintf("dead %1.1f", (p.g.RespawnDelay - time.Since(p.tdeath)).Seconds())
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
			p.screenRunes = nil
			p.ready = false
		}
	}
}

// every tick, based on player screen size - calculate, store and send screen deltas.
func (p *Player) update() {
	if !p.ready {
		return
	}
	g := p.g
	gb := g.board
	// score state
	totalPlayers := len(g.score.allPlayersSorted)
	maxLines := (g.h - 1) - 2         //height units - borders
	maxSlots := maxLines / slotHeight //each player needs 3 lines
	halfSlots := maxSlots / 2
	startIndex := p.index - halfSlots
	if startIndex < 0 {
		startIndex = 0
	}
	// center board (origin) with offset width and height
	ow := (p.w - g.w) / 2
	oh := (p.h - g.h) / 2
	// store the last rendered for network optimisation
	var lastw, lasth uint16
	var r rune
	var c ID
	// screen loop
	var u []byte
	for h := 0; h < g.h; h++ {
		for tw := 0; tw < g.w; tw++ {
			// each iteration draws rune (r) and color (c)
			// at terminal location: w x h
			r = empty
			c = blank
			// choose a rune to draw, either from
			// sidebar or from game board
			if tw < sidebarWidth {
				// pick rune from sidebar
				if tw == 0 {
					r = filled
				} else if h == 0 {
					r = top
				} else if h == g.h-1 {
					r = bottom
				} else {
					bh := h - 1 //borderless height
					playerSlot := bh / slotHeight
					playerIndex := startIndex + playerSlot
					if playerIndex < totalPlayers {
						sp := g.score.allPlayersSorted[playerIndex]
						line := bh % slotHeight
						if tw == 1 {
							switch line {
							case 0:
								sp.score[0] = fmt.Sprintf("%s            ", sp.Name)
							case 1:
								sp.score[1] = fmt.Sprintf("  rank  #%03d  ", sp.rank)
							case 2:
								sp.score[2] = fmt.Sprintf("  %s           ", sp.status())
							case 3:
								sp.score[3] = fmt.Sprintf("  kills %4d   ", sp.Kills)
							}
						}
						if tw-1 < len(sp.score[line]) {
							r = rune(sp.score[line][tw-1])
							c = sp.id
						}
					}
				}
			} else {
				// pick rune from game board, one rune is two game tiles
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
				// choose color (use color of h1, otherwise h2)
				if gb[gw][h2] == blank {
					c = gb[gw][h1]
				} else {
					c = gb[gw][h2]
				}
			}
			// player board is different? draw it
			if p.screenRunes[tw][h] != r ||
				(p.screenRunes[tw][h] != empty && p.screenColors[tw][h] != c) {
				// skip if we only moved one space right
				nexth := uint16(h + 1 + oh)
				nextw := uint16(tw + 1 + ow)
				if nexth != lasth || nextw != lastw+1 {
					u = append(u, ansi.Goto(nexth, nextw)...)
					lasth = nexth
					lastw = nextw
				}
				// p.logf("draw [%d,%d] '%s' (%d)", nexth, nextw, string(r), c)
				// write color
				u = append(u, colours[c]...)
				p.screenColors[tw][h] = c
				// write rune
				u = append(u, []byte(string(r))...)
				p.screenRunes[tw][h] = r
			}
		}
	}
	if len(u) == 0 {
		return
	}
	p.conn.Write(u)
	// p.logf("send %d", len(u))
}
