package rocket

import (
	"log"
    "net/url"
	"sync"

	api "github.com/RocketChat/Rocket.Chat.Go.SDK/realtime"
	models "github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	"github.com/lnxjedi/gopherbot/bot"
)

type config struct {
	Server string // Rocket.Chat server to connect to
	Email string // Rocket.Chat user email
	Password string // the initial userid
}

type rocketConnector struct {
	rt *api.Client
    user *models.User
    running bool
	bot.Handler
    sync.Mutex
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started

func init() {
	bot.RegisterConnector("rocket", Initialize)
}

// Initialize sets up the connector and returns a connector object
func Initialize(robot bot.Handler, l *log.Logger) bot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c config
    var err error

	err = robot.GetProtocolConfig(&c)
	if err != nil {
		robot.Log(bot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}

	cred := &models.UserCredentials {
		Email: c.Email,
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
        rt: client,
        Handler: robot,
    }
    
    if user, err := client.Login(cred); err != nil {
        rc.Log(bot.Fatal, "unable to log in to rocket chat: %v", err)
    } else {
        rc.user = user
    }

	return bot.Connector(rc)
}
