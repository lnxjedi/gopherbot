// Package rocket implements a connector for Rocket.Chat
package rocket

import (
	"sync"

	models "github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	api "github.com/RocketChat/Rocket.Chat.Go.SDK/realtime"
	"github.com/lnxjedi/gopherbot/bot"
)

type config struct {
	Server   string // Rocket.Chat server to connect to
	Email    string // Rocket.Chat user email
	Password string // the initial userid
}

type rocketConnector struct {
	rt      *api.Client
	user    *models.User
	running bool
	bot.Handler
	sync.Mutex
	inChannels  map[string]struct{}
	subChannels map[string]chan<- struct{}
}

var incoming chan *models.Message = make(chan *models.Message)

func (rc *rocketConnector) Run(stop <-chan struct{}) {
	rc.Lock()
	// This should never happen, just a bit of defensive coding
	if rc.running {
		rc.Unlock()
		return
	}
	rc.running = true
	rc.Unlock()
	// TODO: loop on getting messages until stopped
}
