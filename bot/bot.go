// Package bot provides the interfaces for creating a chatops
// bot.
package bot

import (
	"log"
	"regexp"
	"sync"
)

var botLock sync.Mutex
var botCreated bool

// Bot holds all the interal data relevant to the Bot
type Bot struct {
	debug        bool
	alias        rune           // single-char alias for addressing the bot
	name         string         // e.g. "Gort"
	preRegex     *regexp.Regexp // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex    *regexp.Regexp // regex for matching, e.g. "open the pod bay doors, hal"
	channels     []string       // list of channels to join
	sync.RWMutex                // for safe updating of bot data structures
	Connector                   // Connector interface, implemented by each specific protocol
	port         string
}

// interface ChatBot defines the API for plugins
type ChatBot interface {
	GetDebug() bool
	SetDebug(d bool)
	Connector
}

// interface Handler defines the callback API for Connectors
type Handler interface {
	ChannelMsg(channelName, message string)
	DirectMsg(userName, message string)
}

// Instantiate the one and only instance of a Gobot
func Create() *Bot {
	botLock.Lock()
	if botCreated {
		return nil
	}
	botCreated = true
	b := &Bot{}
	botLock.Unlock()
	return b
}

// Print debugging messages if the debug flag is set
func (b *Bot) Debug(v ...interface{}) {
	if b.debug {
		log.Println(v)
	}
}

// Set a one-rune alias for the 'bot'
func (b *Bot) SetAlias(a rune) {
	b.Lock()
	b.alias = a
	b.Unlock()
	b.updateRegexes()
}

// report whether bot debug messages are on or off
func (b *Bot) GetDebug() bool {
	return b.debug
}

// set debugging messages to on or off
func (b *Bot) SetDebug(d bool) {
	b.Lock()
	b.debug = d
	b.Unlock()
}

func (b *Bot) GetInitChannels() []string {
	b.Lock()
	c := b.channels
	b.Unlock()
	return c
}

func (b *Bot) SetInitChannels(ic []string) {
	b.Lock()
	b.channels = ic
	b.Unlock()
}

func (b *Bot) SetName(n string) {
	b.Lock()
	b.Debug("Setting name to: " + n)
	b.name = n
	b.Unlock()
	b.updateRegexes()
}

func (b *Bot) SetPort(p string) {
	b.Lock()
	b.port = p
	b.Unlock()
}

func (b *Bot) Init(c Connector) {
	b.Lock()
	b.Connector = c
	b.Unlock()
	go b.listenHttpJSON()
	for _, channel := range b.GetInitChannels() {
		b.JoinChannel(channel)
	}
	//TODO: remove this later
	name, _ := b.GetProtocolUserAttribute("davidp", "realName")
	b.SendUserMessage("davidp", "Hello, sir! I know who you are now: "+name)
	//	b.conn.SendChannelMessage("C0RK4DG68", "Hello, World!")
}
