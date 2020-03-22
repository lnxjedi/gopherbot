package bot

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/lnxjedi/robot"
)

// func template(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
// 	r := m.(Robot)
// 	return
// }

// rotatelog (task rotate-log); rotate the log file when logging to file
func rotatelog(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	ext := ""
	if len(args) == 1 {
		ext = args[0]
	}
	return logRotate(ext)
}

// logtail - task tail-log; get the last 2k of pipeline log
func logtail(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	hist := w.histName
	idx := w.runIndex
	w.Unlock()
	logReader, err := interfaces.history.GetLog(hist, idx)
	if err != nil && interfaces.history == memHistories {
		Log(robot.Error, "Failed getting log reader in tail-log for history %s, index: %d", hist, idx)
		return robot.Fail
	}
	if err != nil {
		Log(robot.Debug, "Failed getting log reader in tail-log, checking for memlog fallback")
		logReader, err = memHistories.GetLog(hist, idx)
	}
	if err != nil {
		Log(robot.Error, "Failed memlog fallback retrieving %s:%d in tail-log")
		return robot.MechanismFail
	}
	tail := newlineBuffer(2048, 512, "<... truncated...>")
	scanner := bufio.NewScanner(logReader)
	for scanner.Scan() {
		line := scanner.Text()
		tail.writeLine(line)
	}
	tail.close()
	tailReader, _ := tail.getReader()
	buffer, _ := ioutil.ReadAll(tailReader)
	r.Fixed().Say(string(buffer))
	return
}

const maxMailBody = 10485760 // 10MB

// logmail - task email-log; send the job log to one or more email
// addresses.
func logmail(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	if len(args) == 0 {
		m.Log(robot.Error, "email-log called with no addresses")
		return robot.Fail
	}
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	hist := w.histName
	idx := w.runIndex
	w.Unlock()
	logReader, err := interfaces.history.GetLog(hist, idx)
	if err != nil && interfaces.history == memHistories {
		Log(robot.Error, "Failed getting log reader in tail-log for history %s, index: %d", hist, idx)
		return robot.Fail
	}
	if err != nil {
		Log(robot.Debug, "Failed getting log reader in tail-log, checking for memlog fallback")
		logReader, err = memHistories.GetLog(hist, idx)
	}
	if err != nil {
		Log(robot.Error, "Failed memlog fallback retrieving %s:%d in tail-log")
		return robot.MechanismFail
	}
	lr := io.LimitReader(logReader, maxMailBody)
	body := new(bytes.Buffer)
	body.Write([]byte("<pre>\n"))
	buff, rerr := ioutil.ReadAll(lr)
	if rerr != nil {
		r.Log(robot.Error, "Reading history #%d for '%s': %v", idx, hist, rerr)
		return robot.MechanismFail
	}
	body.Write(buff)
	body.Write([]byte("\n</pre>"))
	subject := fmt.Sprintf("Log for pipeline '%s', run %d", hist, idx)
	var ret robot.RetVal = robot.Ok
	for _, addr := range args {
		check := r.EmailAddress(addr, subject, body, true)
		if check != robot.Ok {
			ret = check
		}
	}
	if ret != robot.Ok {
		r.Log(robot.Error, "There was a problem emailing one or more pipeline logs, contact an administrator: %s", ret)
		return robot.Fail
	}
	return
}

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

func quit(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	pn := r.pipeName
	state.Lock()
	if state.shuttingDown {
		state.Unlock()
		Log(robot.Warn, "Quit triggered in pipeline '%s' with shutdown already in progress", pn)
		return
	}
	running := state.pipelinesRunning - 1
	state.shuttingDown = true
	state.Unlock()
	r.Log(robot.Info, "Quit triggered in pipeline '%s' with %d pipelines running (including this one)", pn, running)
	go stop()
	return
}

func pause(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	resume := make(chan struct{})
	brainLocks.Lock()
	brainLocks.locks[w.id] = resume
	brainLocks.Unlock()
	pauseBrain(w.id, resume)
	return
}

func resume(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	brainLocks.Lock()
	if resume, ok := brainLocks.locks[w.id]; ok {
		close(resume)
	}
	brainLocks.Unlock()
	return
}

func init() {
	RegisterTask("restart-robot", true, robot.TaskHandler{Handler: restart})
	RegisterTask("robot-quit", true, robot.TaskHandler{Handler: quit})
	RegisterTask("rotate-log", true, robot.TaskHandler{Handler: rotatelog})
	RegisterTask("pause-brain", true, robot.TaskHandler{Handler: pause})
	RegisterTask("resume-brain", true, robot.TaskHandler{Handler: resume})
	RegisterTask("tail-log", false, robot.TaskHandler{Handler: logtail})
	RegisterTask("email-log", true, robot.TaskHandler{Handler: logmail})
}
