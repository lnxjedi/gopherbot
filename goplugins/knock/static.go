package knock

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPreload("goplugins/knock.so")
	bot.RegisterPlugin("knock", knockhandler)
}
