package dynamobrain

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("brains/dynamodb.so")
	bot.RegisterSimpleBrain("dynamo", provider)
}
