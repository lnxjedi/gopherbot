package rocket

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPreload("connectors/rocket.so")
	bot.RegisterConnector("rocket", Initialize)
}
