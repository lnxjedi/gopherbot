package bot

import "github.com/lnxjedi/gopherbot/robot"

func restart(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	pn := r.pipeName
	state.Lock()
	if state.shuttingDown {
		state.Unlock()
		Log(robot.Warn, "Restart triggered in pipeline '%s' with shutdown already in progress", pn)
		return
	}
	running := state.pipelinesRunning - 1
	state.shuttingDown = true
	state.restart = true
	state.Unlock()
	r.Log(robot.Info, "Restart triggered in pipeline '%s' with %d pipelines running (including this one)", pn, running)
	go stop()
	return
}

func init() {
	RegisterTask("restart", true, robot.TaskHandler{Handler: restart})
}
