// Package gobot provides the functions necessary to
// write chat plugins.
package gobot

import (
	"log"
)

type botListener struct {
	port  string
	owner *Bot
}

var listener botListener

type Bot struct {
	debug bool
	alias string
	name  string
	conn  Connector
}

func New(a string, p string, d bool) *Bot {
	b := &Bot{
		alias: a,
		debug: d}
	listener.owner = b
	listener.port = p
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
	go listener.listenHttpJSON()
	b.conn.SendChannelMessage("C0RK4DG68", "Hello world")
}
