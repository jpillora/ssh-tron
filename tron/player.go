package tron

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/jpillora/ansi"
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

type Player struct {
	id          ID
	x, y        uint8
	d, nextd    Direction
	c           []byte
	row, col    int
	dead, ready bool
	exists      chan bool
	g           *Game
	board       Board //*the player's* view of the board
	conn        *ansi.Ansi
	log         func(string, ...interface{})
}

func NewPlayer(g *Game, conn *net.TCPConn, id ID) *Player {
	//no nagles
	conn.SetNoDelay(true)
	p := &Player{
		id:     id,
		g:      g,
		c:      colours[id],
		dead:   true,
		ready:  false,
		exists: make(chan bool, 1),
		d:      dup,
		conn:   ansi.Wrap(conn),
		log:    log.New(os.Stdout, fmt.Sprintf("player-%d: ", id), 0).Printf,
	}
	return p
}

func (p *Player) respawn() {
	if !p.dead || !p.ready {
		return
	}
	p.x = uint8(rand.Intn(int(p.g.board.width()-2))) + 1
	p.y = uint8(rand.Intn(int(p.g.board.height()-2))) + 1
	p.d = Direction(uint8(rand.Intn(4) + 65))
	p.nextd = p.d
	p.dead = false
}

var charMode = []byte{255, 253, 34, 255, 251, 1}

func (p *Player) play() {
	p.log("connected")

	//put client into character-mode
	p.conn.Write(charMode)
	p.conn.Set(ansi.Reset)
	p.conn.CursorHide()

	go p.resizeWatch()
	go p.recieveActions()

	//block until player quits
	<-p.exists
	p.log("disconnected")
}

func (p *Player) teardown() {
	p.conn.CursorShow()
	p.conn.EraseScreen()
	p.conn.Goto(1, 1)
	p.conn.Set(ansi.Reset)
	p.conn.Close()
	p.exists <- false
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
			p.log("close requested")
			break
		}
		//parse up,down,left,right
		d := byte(p.d)
		if len(b) == 3 && b[0] == ansi.Esc &&
			b[1] == 91 &&
			b[2] >= byte(dup) && b[2] <= byte(dleft) &&
			((d%2 == 0 && d-1 != b[2]) || //while preventing you moving into yourself
				((d+1)%2 == 0 && d+1 != b[2])) {
			p.nextd = Direction(b[2])
			continue
		}
		//respawn!
		if b[0] == 13 {
			p.respawn()
			continue
		}
		// p.log("sent action %+v", b)
	}
	p.teardown()
}

var resizeTmpl = string(ansi.Goto(2, 5)) +
	string(ansi.Set(ansi.White)) +
	"Please resize your terminal to %dx%d (+%dx+%d)"

func (p *Player) resizeWatch() {

	gcol := p.g.board.width()
	grow := p.g.board.termHeight()

	for {
		p.conn.Goto(1000, 1000)
		p.conn.QueryCursorPosition()
		r := <-p.conn.Reports
		if r.Type == ansi.Position {
			if r.Pos.Row >= grow && r.Pos.Col >= gcol {
				//fits
				if !p.ready || p.row != r.Pos.Row || p.col != r.Pos.Col {
					p.log("resize")
					p.row = r.Pos.Row
					p.col = r.Pos.Col
					p.conn.EraseScreen()
					p.board = p.g.board.new()
				}
				p.ready = true
			} else {
				//doesnt fit
				p.conn.EraseScreen()
				p.conn.Write([]byte(fmt.Sprintf(resizeTmpl, gcol, grow,
					int(math.Max(float64(gcol-r.Pos.Col), 0)),
					int(math.Max(float64(grow-r.Pos.Row), 0)))))
				p.board = nil
				p.ready = false
				p.log("not ready")
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

//perform a diff against this players board
//and the game board, sends the result, if any
func (p *Player) update() {
	pb := p.board
	gb := p.g.board

	if !p.ready {
		return
	}

	cols := gb.width()
	rows := gb.termHeight()

	//center board with offset width and height
	ocol := (p.col - cols) / 2
	orow := (p.row - rows) / 2

	var u []byte
	for w := 0; w < cols; w++ {
		for h := 0; h < rows; h++ {
			h1 := h * 2
			h2 := h1 + 1
			if pb[w][h1] != gb[w][h1] || pb[w][h2] != gb[w][h2] {
				var s, c []byte
				//choose rune
				if gb[w][h1] != blank && gb[w][h2] != blank {
					s = full
				} else if gb[w][h1] != blank {
					s = top
				} else if gb[w][h2] != blank {
					s = bot
				} else {
					s = none
				}
				//choose color
				if gb[w][h2] == blank {
					c = colours[gb[w][h1]]
				} else {
					c = colours[gb[w][h2]]
				}

				//player board is different! queue update
				u = append(u, ansi.Goto(uint16(h+1+orow), uint16(w+1+ocol))...)
				//draw it
				u = append(u, c...)
				u = append(u, s...)
				pb[w][h1] = gb[w][h1]
				pb[w][h2] = gb[w][h2]
			}
		}
	}
	if len(u) == 0 {
		return
	}
	p.conn.Write(u)
}
