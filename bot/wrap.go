package bot

import "github.com/lnxjedi/gopherbot/robot/util"

type Wrapper = util.Wrapper

func NewWrapper() Wrapper {
	return util.NewWrapper()
}

func Wrap(s string, limit int) string {
	return util.Wrap(s, limit)
}
