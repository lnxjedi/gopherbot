package bot

import "github.com/lnxjedi/gopherbot/robot"

type Wrapper = robot.Wrapper

func NewWrapper() Wrapper {
	return robot.NewWrapper()
}

func Wrap(s string, limit int) string {
	return robot.Wrap(s, limit)
}
