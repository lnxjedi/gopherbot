// Package gobot provides the interfaces for a Gobot chatops
// bot.
package gobot

import (
	"log"
	"sync"
)

var botLock sync.Mutex
var botCreated bool

type Bot struct {
	debug bool
	alias rune         // single-char alias for addressing the bot
	name  string       // e.g. "Gort"
	lock  sync.RWMutex // for safe updating of bot data structures
	conn  Connector    // Connector interface, implemented by each specific protocol
	port  string
}

// Instantiate the one and only instance of a Gobot
func New() *Bot {
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

// Set a
func (b *Bot) SetAlias(a rune) {
	b.lock.Lock()
	b.alias = a
	b.lock.Unlock()
}

func (b *Bot) GetDebug() bool {
	return b.debug
}

func (b *Bot) SetDebug(d bool) {
	b.lock.Lock()
	b.debug = d
	b.lock.Unlock()
}

func (b *Bot) SetName(n string) {
	b.lock.Lock()
	b.Debug("Setting name to: " + n)
	b.name = n
	b.lock.Unlock()
}

func (b *Bot) SetPort(p string) {
	b.lock.Lock()
	b.port = p
	b.lock.Unlock()
}

func (b *Bot) Init(c Connector) {
	b.lock.Lock()
	b.conn = c
	go b.listenHttpJSON()
	b.lock.Unlock()
	//	b.conn.SendChannelMessage("C0RK4DG68", "Hello, World!")
}
