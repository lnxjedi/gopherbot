// Only needed when built as part of the gopherbot binary
// +build !module

package dynamobrain

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("brains/dynamodb.so")
	bot.RegisterSimpleBrain("dynamo", provider)
}
