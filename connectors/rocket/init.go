package rocket

import (
	"log"
	"net/url"
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
	models "github.com/lnxjedi/gopherbot/connectors/rocket/models"
	api "github.com/lnxjedi/gopherbot/connectors/rocket/realtime"
)

var lock sync.Mutex  // package var lock
var initialized bool // set when connector is initialized

func init() {
	bot.RegisterConnector("rocket", Initialize)
}

// Initialize sets up the connector and returns a connector object
func Initialize(robot bot.Handler, l *log.Logger) bot.Connector {
	lock.Lock()
	if initialized {
		lock.Unlock()
		return nil
	}
	initialized = true
	lock.Unlock()

	var c config
	var err error

	err = robot.GetProtocolConfig(&c)
	if err != nil {
		robot.Log(bot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}

	cred := &models.UserCredentials{
		Email:    c.Email,
		Password: c.Password,
	}

	u, err := url.Parse(c.Server)
	if err != nil {
		robot.Log(bot.Fatal, "Unable to parse URL: %s, %v", c.Server, err)
	}

	client, err := api.NewClient(u, true)
	if err != nil {
		robot.Log(bot.Fatal, "Unable to create client: %v", err)
	}

	rc := &rocketConnector{
		rt:             client,
		Handler:        robot,
		channelNames:   make(map[string]string),
		channelIDs:     make(map[string]string),
		joinedChannels: make(map[string]struct{}),
		dmChannels:     make(map[string]struct{}),
		privChannels:   make(map[string]struct{}),
	}

	if user, err := client.Login(cred); err != nil {
		rc.Log(bot.Fatal, "unable to log in to rocket chat: %v", err)
	} else {
		rc.user = user
	}
	incoming = client.GetMessageStreamUpdateChannel()

	return bot.Connector(rc)
}
