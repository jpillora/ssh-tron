package tron

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

type ID uint16

type Game struct {
	Config
	w, h, bw, bh     int         // total score+board size
	db               *Database   // database
	server           *Server     // ssh server
	score            *scoreboard // state
	bot              *Bot        // chat bot
	board            Board
	idPool           chan ID
	allPlayers       map[string]*Player
	allPlayersSorted []*Player
	currPlayers      map[ID]*Player
	logf             func(format string, args ...interface{})
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
		Config:      c,
		w:           c.Width + sidebarWidth,
		h:           c.Height / 2,
		bw:          c.Height,
		bh:          c.Width,
		db:          db,
		server:      server,
		bot:         &Bot{},
		board:       board,
		idPool:      idPool,
		allPlayers:  make(map[string]*Player),
		currPlayers: make(map[ID]*Player),
		logf:        log.New(os.Stdout, "tron: ", 0).Printf,
	}
	g.score = &scoreboard{g: g}
	//load initial player list
	prevPlayers, err := g.db.loadAll()
	if err != nil {
		return nil, errors.New("Failed to restore player list")
	}
	for _, p := range prevPlayers {
		g.allPlayers[p.hash] = p
	}
	// initialise slack if provided
	if t := c.SlackToken; t != "" {
		ch := c.SlackChannel
		if ch == "" {
			return nil, errors.New("Slack channel must also be specified (--slack-channel)")
		}
		if err := g.bot.init(t, ch); err != nil {
			return nil, err
		}
		motd := "tron server started\n"
		if g.Config.JoinAddress != "" {
			motd += fmt.Sprintf("join using: `ssh %s`", g.Config.JoinAddress)
		} else {
			motd += fmt.Sprintf("join using:\n```\n%s\n```\n", g.server.addresses)
		}
		if err := g.bot.message(motd); err != nil {
			return nil, err
		}
		go g.bot.start()
	}
	//compute initial score, load into slackbot
	g.score.compute()
	//game ready
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
	g.logf("game started (#%d player slots, %s/tick)", len(g.idPool), g.Config.GameSpeed)

	// watch signals (catch Ctrl+C and gracefully shutdown)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go g.watch(c)
	addr := g.Config.JoinAddress
	if addr == "" {
		addr = "\n" + g.server.addresses
	}
	// start the ssh server
	go g.server.start()
	g.logf("server up (fingerprint %s)\njoin at: %s\n", fingerprintKey(g.server.privateKey.PublicKey()), addr)
	// handle incoming players forever (channel never closed)
	for p := range g.server.newPlayers {
		go g.handle(p)
	}
}

func (g *Game) watch(c chan os.Signal) {
	<-c
	g.logf("game ending...")
	for _, p := range g.currPlayers {
		p.teardown()
	}
	g.db.Close()
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)
}

func (g *Game) handle(p *Player) {
	// check not already connected
	if existing, ok := g.allPlayers[p.hash]; ok && existing.id != blank {
		p.teardown()
		p.logf("rejected - already connected as %s", existing.cname)
		g.idPool <- p.id //put back
		return
	}
	// attempt to load previous scores
	if err := g.db.load(p); err != nil {
		//otherwise new player
		g.db.save(p)
	}
	// connected with a valid id
	p.g = g
	g.allPlayers[p.hash] = p
	g.currPlayers[p.id] = p
	g.score.compute()
	// connected
	p.play() //block while playing
	// disconnected
	g.remove(p)
	delete(g.currPlayers, p.id)
	// reinsert back into pool
	g.idPool <- p.id
	p.id = blank
	p.teardown()
}

func (g *Game) death(p *Player) {
	p.Deaths++
	g.score.compute()
	go g.db.save(p) //save new death count
	p.tdeath = time.Now()
	g.remove(p)
}

//time to keep players trail around after death
var deathTrail = 1 * time.Second

func (g *Game) remove(p *Player) {
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
		t0 := time.Now()
		// move each player 1 square
		for _, p := range g.currPlayers {
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
				if other, ok := g.currPlayers[id]; ok && other != p {
					other.Kills++
					g.score.compute()
					go g.db.save(other) //save new kill count
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
		// update bot score list
		if g.score.changed && g.bot.connected {
			g.bot.scoreChange(g.score.allPlayersSorted)
		}
		// send delta updates to each player
		for _, p := range g.currPlayers {
			if p.ready {
				p.update()
			}
		}
		// mark score as used
		g.score.changed = false
		// game sleep! (attempt to stablize game speed)
		cpu := time.Now().Sub(t0)
		time.Sleep(g.GameSpeed - cpu*2)
	}
}
