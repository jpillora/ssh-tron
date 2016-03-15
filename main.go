package main

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/jpillora/opts"
	"github.com/jpillora/ssh-tron/tron"
)

var VERSION = "0.0.0-src"

func main() {

	c := tron.Config{
		Port:       2200,
		Width:      60,
		Height:     60,
		MaxPlayers: 6,
		// Mode:         "kd",
		// KickDeaths:   5,
		GameSpeed:    40 * time.Millisecond,
		RespawnDelay: 2 * time.Second,
		DBLocation:   filepath.Join(os.TempDir(), "tron.db"),
	}

	opts.New(&c).
		PkgRepo().
		Version(VERSION).
		Parse()

	rand.Seed(time.Now().UnixNano())

	g, err := tron.NewGame(c)
	if err != nil {
		log.Fatal(err)
	}
	g.Play()
}
