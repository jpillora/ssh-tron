//main creates and 'Play's
//a single game instance
package tron

import (
	"log"
	"os"
	"os/signal"
	"time"
)

const maxdeaths = 10

type ID uint16

type Game struct {
	//settings
	maxplayers, speed int
	//total score+board size
	w, h, bw, bh int
	//ssh server
	server *Server
	//state
	sidebar  *Sidebar
	board    Board
	idPool   chan ID
	playerId ID
	players  map[ID]*Player
	log      func(string, ...interface{})
}

func NewGame(port, width, height, maxplayers, speed int) *Game {

	//create an id pool
	idPool := make(chan ID, maxplayers)
	for id := 1; id <= maxplayers; id++ {
		idPool <- ID(id)
	}

	g := &Game{
		maxplayers, speed,
		width + sidebarWidth, height / 2, width, height,
		NewServer(port, idPool),
		nil,
		NewBoard(uint8(width), uint8(height)),
		idPool, 0, map[ID]*Player{},
		log.New(os.Stdout, "tron: ", 0).Printf,
	}

	g.sidebar = NewSidebar(g)

	return g
}

func (g *Game) Play() {

	//build walls
	for w := 0; w < g.bw; w++ {
		g.board[w][0] = wall
		g.board[w][g.bh-1] = wall
	}
	for h := 0; h < g.bh; h++ {
		g.board[0][h] = wall
		g.board[g.bw-1][h] = wall
	}

	//start the game ticker!
	go g.tick()

	//ready for players!
	g.log("game started (#%d player slots)", len(g.idPool))

	//watch signals (catch Ctrl+C and gracefully shutdown)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go g.watch(c)

	//start the ssh server
	go g.server.start()

	//handle incoming players
	for p := range g.server.newPlayers {
		go g.handle(p)
	}
}

func (g *Game) watch(c chan os.Signal) {
	<-c
	for _, p := range g.players {
		p.teardown()
	}
	g.log("game ending...")
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)
}

func (g *Game) handle(p *Player) {
	//connected with a valid id
	//set game and id then play (blocking)
	p.g = g
	g.players[p.id] = p
	g.sidebar.render()
	p.play()
	//disconnected
	delete(g.players, p.id)
	g.death(p)
	//reinsert back into pool
	g.idPool <- p.id
	g.sidebar.render()
}

func (g *Game) death(p *Player) {
	p.waiting = true
	p.deaths++

	//render on score change
	g.sidebar.render()

	time.Sleep(2 * time.Second)
	for w := 0; w < g.bw; w++ {
		for h := 0; h < g.bh; h++ {
			if g.board[w][h] == p.id {
				g.board[w][h] = blank
			}
		}
	}
	p.waiting = false

	//maximum deaths! kick!
	if p.deaths == maxdeaths {
		p.teardown()
	}
}

func (g *Game) tick() {
	//forever
	for {
		//move each player 1 square
		for _, p := range g.players {
			//skip this player
			if p.dead {
				continue
			}
			//move player in [d]irection
			p.d = p.nextd
			switch p.d {
			case dup:
				p.y--
			case ddown:
				p.y++
			case dleft:
				p.x--
			case dright:
				p.x++
			}
			//player is in a wall
			if g.board[p.x][p.y] != blank {
				//is it another player's wall? kills++
				id := g.board[p.x][p.y]
				if other, ok := g.players[id]; ok && other != p {
					other.kills++
					other.log("killed %s", p.cname)
				}
				//this player dies...
				p.dead = true
				go g.death(p)
				continue
			}
			//place a player square
			g.board[p.x][p.y] = p.id
		}
		//send delta updates to each player
		for _, p := range g.players {
			if p.ready {
				p.update()
			}
		}
		//mark update sent to all
		g.sidebar.changed = false
		//sleep
		time.Sleep(time.Duration(g.speed) * time.Millisecond)
	}
}
