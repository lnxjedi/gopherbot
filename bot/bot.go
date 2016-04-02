package gobot

import (
	"log"
)

type Bot struct {
	debug bool
	alias string
	name  string
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
