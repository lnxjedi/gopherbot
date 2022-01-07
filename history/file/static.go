package filehistory

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterHistoryProvider("file", provider)
}
