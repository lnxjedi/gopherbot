package bot

import (
	"bufio"
	"io/ioutil"

	"github.com/lnxjedi/robot"
)

// logbuffers.go - utility functions for pulling pipeline logs in to
// tail or email buffers.

const maxMailBody = 10485760 // 10MB
const maxMailLine = 16384    // 16k lines
const tailBody = 3072        // 3k total buffer
const tailLine = 512         // max line length for tail

func getLogTail(tag string, idx int) (ret robot.TaskRetVal, buff []byte) {
	ret, buff = getLogBuffer(tag, " ...", idx, tailBody, tailLine)
	return
}

func getLogMail(tag string, idx int) (ret robot.TaskRetVal, buff []byte) {
	ret, buff = getLogBuffer(tag, "< ... truncated>", idx, maxMailBody, maxMailLine)
	return
}

func getLogBuffer(tag, trunc string, idx, buffsize, linesize int) (ret robot.TaskRetVal, buff []byte) {
	logReader, err := interfaces.history.GetLog(tag, idx)
	if err != nil && interfaces.history == memHistories {
		Log(robot.Error, "Failed getting log reader in tail-log for history %s, index: %d", tag, idx)
		ret = robot.NotFound
		return
	}
	if err != nil {
		Log(robot.Debug, "Failed getting log reader in tail-log, checking for memlog fallback")
		logReader, err = memHistories.GetLog(tag, idx)
	}
	if err != nil {
		Log(robot.Error, "Failed memlog fallback retrieving %s:%d in tail-log")
		ret = robot.NotFound
		return
	}
	tail := newLineBuffer(buffsize, linesize, trunc)
	scanner := bufio.NewScanner(logReader)
	for scanner.Scan() {
		line := scanner.Text()
		tail.writeLine(line)
	}
	tail.close()
	tailReader, _ := tail.getReader()
	buff, _ = ioutil.ReadAll(tailReader)

	return
}
