package tron

import "time"

type Config struct {
	Port         int           `help:"Port to listen for TCP connections on"`
	Width        int           `help:"Width of the game world" min:"32" max:"256"`
	Height       int           `help:"Height of the game world" min:"32" max:"256"`
	MaxPlayers   int           `help:"Maximum number of simultaneous players"`
	GameSpeed    time.Duration `help:"Game tick interval, basically controls how fast each player moves"`
	RespawnDelay time.Duration `help:"The time a player must wait before being able to respawn"`
	DBLocation   string        `help:"Location of tron.db, stores game score and config"`
	DBReset      bool          `help:"Reset all scores in the database"`
}

// TODO
// KickDeaths   int           `help:"Punish bad players by kicking them out after N deaths in a row"`
// Mode         string        `help:"Maximum number of deaths before being kicked"`
