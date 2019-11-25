// Only needed when built as part of the gopherbot binary
// +build !module

package rocket

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("connectors/rocket.so")
	bot.RegisterConnector("rocket", Initialize)
}
