package tron

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"time"
)

type ID uint16

type Game struct {
	Config
	w, h, bw, bh int       // total score+board size
	db           *Database // database
	server       *Server   // ssh server
	sidebar      *sidebar  // state
	board        Board
	idPool       chan ID
	playerID     ID
	players      map[ID]*Player
	logf         func(format string, args ...interface{})
}

// NewGame returns an initialized Game according to the input arguments.
// The main() function should call the Play() method on this Game.
func NewGame(c Config) (*Game, error) {
	if c.Height < 32 || c.Height > 255 {
		return nil, errors.New("height must be between 32-256")
	}
	if c.Width < 32 || c.Width > 255 {
		return nil, errors.New("width must be between 32-256")
	}
	db, err := NewDatabase(c.DBLocation, c.DBReset)
	if err != nil {
		return nil, err
	}

	board, err := NewBoard(uint8(c.Width), uint8(c.Height))
	if err != nil {
		return nil, err
	}

	// create an id pool
	idPool := make(chan ID, c.MaxPlayers)
	for id := 1; id <= c.MaxPlayers; id++ {
		idPool <- ID(id)
	}

	server, err := NewServer(db, c.Port, idPool)
	if err != nil {
		return nil, err
	}

	g := &Game{
		Config:   c,
		w:        c.Width + sidebarWidth,
		h:        c.Height / 2,
		bw:       c.Height,
		bh:       c.Width,
		db:       db,
		server:   server,
		sidebar:  nil,
		board:    board,
		idPool:   idPool,
		playerID: 0,
		players:  make(map[ID]*Player),
		logf:     log.New(os.Stdout, "tron: ", 0).Printf,
	}

	// sidebar height - top and bottom rows are borders
	h := g.h - 2
	g.sidebar = &sidebar{
		g:      g,
		height: h,
		runes:  make([][]rune, h),
	}

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
	signal.Notify(c, os.Interrupt, os.Kill)
	go g.watch(c)

	// start the ssh server
	go g.server.start()

	// handle incoming players forever (channel never closed)
	for p := range g.server.newPlayers {
		go g.handle(p)
	}
}

func (g *Game) watch(c chan os.Signal) {
	<-c
	g.logf("game ending...")
	for _, p := range g.players {
		p.teardown()
	}
	g.db.Close()
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)
}

func (g *Game) handle(p *Player) {
	// attempt to load previous scores
	g.db.LoadPlayer(p)
	// connected with a valid id
	p.g = g
	g.players[p.id] = p
	// connected
	p.play() //block while playing
	// disconnected
	delete(g.players, p.id)
	g.died(p)
	// reinsert back into pool
	g.idPool <- p.id
}

func (g *Game) death(p *Player) {
	p.Deaths++
	go g.db.SavePlayer(p) //save new death count
	p.tdeath = time.Now()
	// maximum deaths! kick!
	if p.Deaths == g.KickDeaths {
		p.teardown()
	}
	g.died(p)
}

//time to keep players trail around after death
var deathTrail = 1 * time.Second

func (g *Game) died(p *Player) {
	p.waiting = true

	//respawn/deathtrail time
	if g.RespawnDelay > deathTrail {
		time.Sleep(deathTrail)
	} else {
		time.Sleep(g.RespawnDelay)
	}
	// clear this player off the board!
	for w := 0; w < g.bw; w++ {
		for h := 0; h < g.bh; h++ {
			if g.board[w][h] == p.id {
				g.board[w][h] = blank
			}
		}
	}
	//respawn extra
	if g.RespawnDelay > deathTrail {
		time.Sleep(g.RespawnDelay - deathTrail)
	}

	p.waiting = false
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
					other.Kills++
					go g.db.SavePlayer(other) //save new kill count
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
		// game sleep!
		time.Sleep(g.GameSpeed)
	}
}
