package filehistory

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterHistoryProvider("file", provider)
}
