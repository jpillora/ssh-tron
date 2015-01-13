//main creates and 'Play's
//a single game instance
package tron

import (
	"log"
	"os"
	"os/signal"
	"time"
)

type ID uint16

type Game struct {
	maxplayers, maxdeaths int           // settings
	speed, delay          time.Duration // game speed and respawn delay
	w, h, bw, bh          int           // total score+board size
	server                *Server       // ssh server
	sidebar               *Sidebar      // state
	board                 Board
	idPool                chan ID
	playerId              ID
	players               map[ID]*Player
	logf                  func(format string, vars ...interface{})
}

func NewGame(port, width, height, maxplayers, maxdeaths int, speed, delay time.Duration) (*Game, error) {

	// create an id pool
	idPool := make(chan ID, maxplayers)
	for id := 1; id <= maxplayers; id++ {
		idPool <- ID(id)
	}

	board, err := NewBoard(uint8(width), uint8(height))
	if err != nil {
		return nil, err
	}

	server, err := NewServer(port, idPool)
	if err != nil {
		return nil, err
	}

	g := &Game{
		maxplayers, maxdeaths,
		speed,
		delay,
		width + sidebarWidth, height / 2, width, height,
		server,
		nil,
		board,
		idPool, 0, map[ID]*Player{},
		log.New(os.Stdout, "tron: ", 0).Printf,
	}

	g.initSidebar()

	return g, nil
}

func (g *Game) Play() {

	// build walls
	for w := 0; w < g.bw; w++ {
		g.board[w][0] = wall
		g.board[w][g.bh-1] = wall
	}
	for h := 0; h < g.bh; h++ {
		g.board[0][h] = wall
		g.board[g.bw-1][h] = wall
	}

	// start the game ticker!
	go g.tick()

	// ready for players!
	g.logf("game started (#%d player slots)", len(g.idPool))

	// watch signals (catch Ctrl+C and gracefully shutdown)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go g.watch(c)

	// start the ssh server
	go g.server.start()

	// handle incoming players
	for p := range g.server.newPlayers {
		go g.handle(p)
	}
}

func (g *Game) watch(c chan os.Signal) {
	<-c
	for _, p := range g.players {
		p.teardown()
	}
	g.logf("game ending...")
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)
}

func (g *Game) handle(p *Player) {
	// connected with a valid id
	// set game and id then play (blocking)
	p.g = g
	g.players[p.id] = p
	p.play()
	// disconnected
	delete(g.players, p.id)
	g.death(p)
	// reinsert back into pool
	g.idPool <- p.id
}

func (g *Game) death(p *Player) {
	p.waiting = true
	p.deaths++
	p.tdeath = time.Now()

	// leave player on board for [delay]
	time.Sleep(g.delay)

	// clear!
	for w := 0; w < g.bw; w++ {
		for h := 0; h < g.bh; h++ {
			if g.board[w][h] == p.id {
				g.board[w][h] = blank
			}
		}
	}

	p.waiting = false

	// maximum deaths! kick!
	if p.deaths == g.maxdeaths {
		p.teardown()
	}
}

func (g *Game) tick() {
	// loop forever
	for {
		// move each player 1 square
		for _, p := range g.players {
			// skip this player
			if p.dead {
				continue
			}
			// move player in [d]irection
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
			// player is in a wall
			if g.board[p.x][p.y] != blank {
				// is it another player's wall? kills++
				id := g.board[p.x][p.y]
				if other, ok := g.players[id]; ok && other != p {
					other.kills++
					other.logf("killed %s", p.cname)
				}
				// this player dies...
				p.dead = true
				go g.death(p)
				continue
			}
			// place a player square
			g.board[p.x][p.y] = p.id
		}

		// render the sidebar (and potentially flip the changed flag)
		g.sidebar.render()

		// send delta updates to each player
		for _, p := range g.players {
			if p.ready {
				p.update()
			}
		}
		// mark update sent to all
		g.sidebar.changed = false
		// sleep
		time.Sleep(g.speed)
	}
}
