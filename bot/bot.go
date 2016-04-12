// Package gobot provides the functions necessary to
// write chat plugins.
package gobot

import (
	"log"
	"sync"
)

var botLock sync.Mutex
var botCreated bool

type botListener struct {
	port  string
	owner *Bot
}

var listener botListener

type Bot struct {
	debug bool
	alias string
	name  string
	port  string
	conn  Connector
}

func New(a string, p string, d bool) *Bot {
	botLock.Lock()
	if botCreated {
		return nil
	}
	botCreated = true
	b := &Bot{
		alias: a,
		debug: d,
		port:  p,
	}
	botLock.Unlock()
	return b
}

func (b *Bot) Debug(v ...interface{}) {
	if b.debug {
		log.Println(v)
	}
}

func (b *Bot) GetDebug() bool {
	return b.debug
}

func (b *Bot) SetName(n string) {
	b.Debug("Setting name to:" + n)
	b.name = n
}

func (b *Bot) Init(c Connector) {
	b.conn = c
	go b.listenHttpJSON()
	//	b.conn.SendChannelMessage("C0RK4DG68", "Hello, World!")
}
