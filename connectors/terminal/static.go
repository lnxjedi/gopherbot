// Only needed when built as part of the gopherbot binary
// +build !module

package terminal

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("connectors/terminal.so")
	bot.RegisterConnector("terminal", Initialize)
}
