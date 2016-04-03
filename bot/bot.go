// Package gobot provides the functions necessary to
// write chat plugins.
package gobot

import (
	"log"
)

type Bot struct {
	debug bool
	alias string
	name  string
	port  string
	conn  Connector
}

func New(a string, d bool) *Bot {
	return &Bot{
		alias: a,
		debug: d}
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

func (b *Bot) SetHttpPort(p string) {
	b.port = p
}

func (b *Bot) Init(c Connector) {
	b.conn = c
	go b.ListenHttpJSON()
	b.conn.SendChannelMsg("C0RK4DG68", "Hello world")
}
