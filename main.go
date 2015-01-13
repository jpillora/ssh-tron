package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/jpillora/ssh-tron/tron"
)

var (
	port       = flag.Int("port", 2200, "Port to listen for TCP connections on")
	width      = flag.Int("width", 60, "Width of the game world")
	height     = flag.Int("height", 60, "Height of the game world")
	maxplayers = flag.Int("players", 4, "Maximum number of simultaneous players")
	maxdeaths  = flag.Int("deaths", 10, "Maximum number of deaths before being kicked")
	speed      = flag.Duration("speed", 25*time.Millisecond, "Game tick interval")
	delay      = flag.Duration("delay", 2*time.Second, "Respawn delay")
)

func main() {
	flag.Parse()

	if *height < 32 {
		log.Fatal(`'height' must be at least 32`)
	}

	if *width < 32 {
		log.Fatal(`'width' must be atleast 32`)
	}

	rand.Seed(time.Now().UnixNano())

	g, err := tron.NewGame(*port, *width, *height, *maxplayers, *maxdeaths, *speed, *delay)
	if err != nil {
		log.Fatal(err)
	}
	g.Play()
}
