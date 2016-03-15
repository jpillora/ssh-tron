package tron

import (
	"fmt"
	"log"
	"regexp"

	"github.com/nlopes/slack"
)

type Bot struct {
	api     *slack.Client
	channel string
	top     *Player
	scores  string
}

func (b *Bot) init(token, channel string) error {
	b.api = slack.New(token)
	b.channel = channel
	resp, err := b.api.AuthTest()
	if err != nil {
		return err
	}
	log.Printf("authenticated on slack as: %s", resp.User)
	return nil
}

func (b *Bot) message(msg string) error {
	if _, _, err := b.api.PostMessage("#"+b.channel, msg, slack.PostMessageParameters{AsUser: true}); err != nil {
		return err //fmt.Errorf("failed to send slack message to channel: %s: %s", channel, err)
	}
	return nil
}

var scoresRe = regexp.MustCompile(`(?i)tron\s*scores?\b`)

func (b *Bot) scoreChange(ps []*Player) {
	if len(ps) > 5 {
		ps = ps[:5]
	}
	var top *Player
	b.scores = ""
	for i, p := range ps {
		if i == 0 && p.Kills > 0 {
			top = p
		}
		b.scores += fmt.Sprintf("#%d *%s* `%d` kills\n", i+1, p.sshname, p.Kills)
	}
	if top != nil && b.top != top && (b.top == nil || top.Kills > b.top.Kills) {
		b.message("*" + top.sshname + "* has taken the lead!\n\n" + b.scores)
		b.top = top
	}
}

func (b *Bot) start() {
	rtm := b.api.NewRTM()
	go rtm.ManageConnection()
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				// log.Printf("%s, %s, %s", ev.Channel, ev.Text, ev.User)
				if scoresRe.MatchString(ev.Text) {
					b.message(b.scores)
				}
			case *slack.RTMError:
				log.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				log.Printf("Invalid slack credentials")
				return
			}
		}
	}
}
