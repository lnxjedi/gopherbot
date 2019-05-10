// Package rocket implements a connector for Rocket.Chat
package rocket

// import (
// 	"fmt"
// 	"strings"
// 	"sync"

// 	"github.com/lnxjedi/gopherbot/bot"
// )

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
