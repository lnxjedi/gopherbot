package rocket

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterConnector("rocket", Initialize)
}
