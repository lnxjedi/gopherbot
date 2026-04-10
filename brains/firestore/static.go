package firestorebrain

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterSimpleBrain("firestore", provider)
}
