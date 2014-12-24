package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/jpillora/tron/tron"
)

var port = flag.Int("port", 2200, "Port to listen for TCP connections on")
var width = flag.Int("width", 80, "Width of the game world")
var height = flag.Int("height", 80, "Height of the game world")
var maxplayers = flag.Int("players", 6, "Maximum number of simultaneous players")
var maxdeaths = flag.Int("deaths", 10, "Maximum number of deaths before being kicked")
var speed = flag.Int("speed", 25, "Game tick interval (in ms)")
var delay = flag.Int("delay", 2000, "Respawn delay (in ms)")

func main() {
	flag.Parse()

	if *height < 32 || *width < 32 {
		log.Fatal("'width' and 'height' must be at least 32")
	}

	rand.Seed(time.Now().UnixNano())
	tron.NewGame(*port, *width, *height, *maxplayers, *maxdeaths, *speed, *delay).Play()
}
