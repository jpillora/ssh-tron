package main

import (
	"flag"
	"math/rand"
	"time"

	"github.com/lumanetworks/telnet-tron/tron"
)

var port = flag.Int("port", 3000, "Port to listen for TCP connections on")
var width = flag.Int("width", 80, "Width of the game world")
var height = flag.Int("height", 80, "Height of the game world")
var maxplayers = flag.Int("max", 6, "Maximum numbers of simultaneous players")
var speed = flag.Int("speed", 25, "Game tick interval (in ms)")

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	tron.NewGame(*port, *width, *height, *maxplayers, *speed).Play()
}
