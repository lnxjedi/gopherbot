package rocket

import (
	"crypto/md5"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/bot"
	models "github.com/lnxjedi/gopherbot/connectors/rocket/models"
	api "github.com/lnxjedi/gopherbot/connectors/rocket/realtime"
)

var lock sync.Mutex  // package var lock
var initialized bool // set when connector is initialized

type config struct {
	Server   string // Rocket.Chat server to connect to
	Email    string // Rocket.Chat user email
	Password string // the initial userid
}

// type and vars for tracking duplicate incoming messages
type msgTrack struct {
	msgID   string
	msgHash [md5.Size]byte
}

type msgQuery struct {
	reply chan bool
	msgTrack
}

// How long to track an incoming message for duplicates
const msgExpire = 7 * time.Second

var trackedMsgs = make(map[msgTrack]time.Time)
var check = make(chan msgQuery)

var userName, userID string

type rocketConnector struct {
	rt      *api.Client
	running bool
	bot.Handler
	sync.RWMutex
	channelNames    map[string]string   // map from roomID to channel name
	channelIDs      map[string]string   // map from channel name to roomID
	joinedChannels  map[string]struct{} // channels we've joined
	dmChannels      map[string]struct{} // direct message channels
	privChannels    map[string]struct{} // private channels
	userNameIDMap   map[string]string   // map of rocket.chat username to userID
	userIDNameMap   map[string]string   // map of userID to username
	gbuserNameIDMap map[string]string   // configured map of username to userID
	gbuserIDNameMap map[string]string   // configured map, iD to configured username
	userDM          map[string]string   // map from username to dm roomID
}

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
		userNameIDMap:  make(map[string]string),
		userIDNameMap:  make(map[string]string),
		// gbuserMap should remain nil until set
		userDM: make(map[string]string),
	}

	if user, err := client.Login(cred); err != nil {
		rc.Log(bot.Fatal, "unable to log in to rocket chat: %v", err)
	} else {
		// NOTE: the login user object doesn't have the UserName
		//userName = user.UserName
		userID = user.ID
		robot.SetBotID(user.ID)
		//robot.SetBotMention(user.UserName)
	}
	incoming = client.GetMessageStreamUpdateChannel()
	return bot.Connector(rc)
}
