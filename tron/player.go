package tron

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/lumanetworks/ansi"
)

type Direction byte

const (
	dup Direction = iota + 65
	ddown
	dright
	dleft
)

type Player struct {
	id          ID
	x, y        uint8
	d           Direction
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
		dead:   true,
		ready:  false,
		exists: make(chan bool, 1),
		d:      dup,
		board:  NewBoard(g.width, g.height),
		conn:   ansi.Wrap(conn),
		log:    log.New(os.Stdout, fmt.Sprintf("player-%d: ", id), 0).Printf,
	}
	return p
}

func (p *Player) respawn() {
	if !p.dead || !p.ready {
		return
	}
	p.x = uint8(rand.Intn(int(p.g.width-2))) + 1
	p.y = uint8(rand.Intn(int(p.g.height-2))) + 1
	p.d = Direction(uint8(rand.Intn(4) + 65))
	p.dead = false
}

func (p *Player) play() {
	p.log("connected")
	go p.recieveActions()
	p.setup()
	<-p.exists
	p.log("disconnected")
}

var charMode = []byte{255, 253, 34, 255, 251, 1}

func (p *Player) setup() {

	col := p.board.Width()
	row := p.board.TermHeight()

	//put client into character-mode
	p.conn.Write(charMode)
	p.conn.Set(ansi.Reset)
	p.conn.CursorHide()

	//perform ready check
	p.conn.Goto(1, 1)
	p.conn.EraseScreen()
	p.conn.Write([]byte(fmt.Sprintf(
		"\r\n"+
			"   Please resize your terminal to %dx%d\r\n"+
			"   and then line up the top edge with\r\n"+
			"   the *very top* of your terminal\r\n"+
			"     [This text should not be visibile]\r\n", col, row)))

	for {
		p.conn.Goto(1000, 1000)
		p.conn.QueryCursorPosition()
		r := <-p.conn.Reports
		if r.Type == ansi.Position {
			if r.Pos.Row >= row && r.Pos.Col >= col {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	p.conn.EraseScreen()
	p.ready = true
	p.log("ready")
}

func (p *Player) teardown() {
	p.conn.CursorShow()
	p.conn.EraseScreen()
	p.conn.Goto(1, 1)
	p.conn.Set(ansi.Reset)
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
			p.teardown()
			p.conn.Close()
			break
		}
		//parse up,down,left,right
		d := byte(p.d)
		if len(b) == 3 && b[0] == ansi.Esc &&
			b[1] == 91 &&
			b[2] >= byte(dup) && b[2] <= byte(dleft) &&
			((d%2 == 0 && d-1 != b[2]) ||
				((d+1)%2 == 0 && d+1 != b[2])) {
			p.d = Direction(b[2])
			continue
		}
		//respawn!
		if b[0] == 13 {
			p.respawn()
			continue
		}
		p.log("sent %+v", b)
	}
}

//perform a diff against this players board
//and the game board, sends the result, if any
func (p *Player) update() {
	pb := p.board
	gb := p.g.board
	var u []byte
	for w := uint8(0); w < p.g.width; w++ {
		for h := uint8(0); h < p.g.height/2; h++ {
			h1 := h * 2
			h2 := h1 + 1
			if pb[w][h1] != gb[w][h1] || pb[w][h2] != gb[w][h2] {
				var wall string
				if gb[w][h1] != blank && gb[w][h2] != blank {
					wall = full
				} else if gb[w][h1] != blank {
					wall = top
				} else if gb[w][h2] != blank {
					wall = bot
				} else {
					wall = none
				}
				//player board is different! queue update
				u = append(u, ansi.Goto(uint16(h+1), uint16(w+1))...)
				u = append(u, []byte(wall)...)
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
