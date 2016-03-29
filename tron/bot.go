package tron

import (
	"fmt"
	"log"
	"regexp"

	"github.com/nlopes/slack"
)

const topNumPlayers = 10

type Bot struct {
	connected bool
	api       *slack.Client
	channel   string
	top       *Player
	scores    string
}

func (b *Bot) init(token, channel string) error {
	b.api = slack.New(token)
	b.channel = channel
	resp, err := b.api.AuthTest()
	if err != nil {
		return err
	}
	fmt.Printf("authenticated on slack as: %s\n", resp.User)
	b.connected = true
	return nil
}

func (b *Bot) message(msg string) error {
	return b.messageTo("#"+b.channel, msg)
}

func (b *Bot) messageTo(to, msg string) error {
	if _, _, err := b.api.PostMessage(to, msg, slack.PostMessageParameters{AsUser: true}); err != nil {
		log.Printf("failed to send slack message to: %s: %s", to, err)
		return err
	}
	return nil
}

var scoresRe = regexp.MustCompile(`(?i)tron\s*scores?\b`)

func (b *Bot) scoreChange(ps []*Player) {
	if len(ps) > topNumPlayers {
		ps = ps[:topNumPlayers]
	}
	var top *Player
	b.scores = ""
	//keep rendered string of scores
	for i, p := range ps {
		if i == 0 && p.Kills > 0 {
			top = p
		}
		b.scores += fmt.Sprintf("#%d *%s* `%d` kills\n", p.rank, p.SSHName, p.Kills)
	}
	//if leader changed, send message
	if top != nil && b.top != top && (b.top == nil || top.rank > b.top.rank) {
		b.message("*" + top.SSHName + "* has taken the lead!\n\n" + b.scores)
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
					if ch, err := b.api.GetChannelInfo(ev.Channel); err == nil {
						b.messageTo("#"+ch.Name, b.scores)
					} else if us, err := b.api.GetUserInfo(ev.User); err == nil {
						b.messageTo("@"+us.Name, b.scores)
					} else {
						b.message(b.scores)
					}
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
