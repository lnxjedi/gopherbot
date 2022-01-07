package knock

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPlugin("knock", knockhandler)
}
