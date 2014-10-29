package tron

import (
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/jpillora/ansi"
)

type ID uint16

var matchip = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+`)

type Game struct {
	//settings
	port, maxplayers, speed int
	width, height           uint8
	//state
	board    Board
	playerId ID
	players  map[ID]*Player
	log      func(string, ...interface{})
}

func NewGame(port, width, height, maxplayers, speed int) *Game {
	return &Game{
		port, maxplayers, speed,
		uint8(width), uint8(height),
		NewBoard(uint8(width), uint8(height)),
		0, map[ID]*Player{},
		log.New(os.Stdout, "server: ", 0).Printf,
	}
}

func (g *Game) Play() {
	//bind to port
	server, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: g.port})
	if err != nil {
		log.Fatal(err)
	}
	//initialise the game board
	g.initialise()
	//start the ticker!
	go g.tick()
	//ready for players!
	g.log("%stelnet-tron%s", ansi.Set(ansi.Green), ansi.Set(ansi.Reset))
	g.log(" join at:")
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipv4 := matchip.FindString(a.String())
		if ipv4 != "" {
			g.log("  ○ telnet %s %d", ipv4, g.port)
		}
	}
	//watch signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go g.watch(c)
	//accept all
	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			continue
		}
		go g.handle(conn)
	}
}

func (g *Game) watch(c chan os.Signal) {
	<-c
	for _, p := range g.players {
		p.teardown()
	}
	g.log("Closing server...")
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)
}

func (g *Game) initialise() {
	//build walls
	for w := uint8(0); w < g.width; w++ {
		g.board[0][w] = wall
		g.board[g.height-1][w] = wall
	}
	for h := uint8(0); h < g.height; h++ {
		g.board[h][0] = wall
		g.board[h][g.width-1] = wall
	}
}

func (g *Game) handle(conn *net.TCPConn) {
	//choose an id
	id := ID(0)
	taken := true
	for taken {
		id++
		_, taken = g.players[id]
		if int(id) > g.maxplayers {
			conn.Write([]byte("This game is full."))
			conn.Close()
			return
		}
	}
	//connected
	p := NewPlayer(g, conn, id)
	g.players[id] = p
	p.play()
	//disconnected
	delete(g.players, id)
	g.death(p.id)
}

func (g *Game) death(pid ID) {
	for w := uint8(0); w < g.width; w++ {
		for h := uint8(0); h < g.height; h++ {
			if g.board[w][h] == pid {
				g.board[w][h] = blank
			}
		}
	}
}

func (g *Game) tick() {
	for _, p := range g.players {
		//skip this player
		if p.dead {
			continue
		}
		//move player in [d]irection
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
			p.dead = true
			g.death(p.id)
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
	time.Sleep(time.Duration(g.speed) * time.Millisecond)
	g.tick()
}
