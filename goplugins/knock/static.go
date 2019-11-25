// +build !module

package knock

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("goplugins/knock.so")
	bot.RegisterPlugin("knock", knockhandler)
}
