package bot

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/modules/linebuffer"
	"golang.org/x/sys/unix"
)

const (
	livePipelineLogBufferSize = 64 * 1024
	livePipelineLogLineSize   = 2048
	livePipelineLogTruncated  = "<... truncated>"
)

type pipelineLiveLogger struct {
	base       robot.HistoryLogger
	live       *linebuffer.Buffer
	mu         sync.Mutex
	baseClosed bool
}

func newPipelineLiveLogger(base robot.HistoryLogger) *pipelineLiveLogger {
	return &pipelineLiveLogger{
		base: base,
		live: linebuffer.New(livePipelineLogBufferSize, livePipelineLogLineSize, livePipelineLogTruncated),
	}
}

func (l *pipelineLiveLogger) Log(line string) {
	tsLine := fmt.Sprintf("%s %s", time.Now().Format("Jan 2 15:04:05"), line)
	l.live.WriteLine(tsLine)
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.baseClosed {
		l.base.Log(line)
	}
}

func (l *pipelineLiveLogger) Line(line string) {
	l.live.WriteLine(line)
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.baseClosed {
		l.base.Line(line)
	}
}

func (l *pipelineLiveLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.baseClosed {
		return
	}
	l.baseClosed = true
	l.base.Close()
}

func (l *pipelineLiveLogger) Finalize() {
	l.live.Close()
	l.mu.Lock()
	baseClosed := l.baseClosed
	if !l.baseClosed {
		l.baseClosed = true
	}
	l.mu.Unlock()
	if !baseClosed {
		l.base.Close()
	}
	l.base.Finalize()
}

func (l *pipelineLiveLogger) Snapshot() io.Reader {
	return l.live.Snapshot()
}

func determinePipelineOperatorChannel(cfg *configuration, task *Task, isJob bool) string {
	if cfg == nil {
		return ""
	}
	if isJob {
		return strings.TrimSpace(task.Channel)
	}
	return strings.TrimSpace(cfg.defaultJobChannel)
}

func formatPipelineClock(ts time.Time, loc *time.Location) string {
	if ts.IsZero() {
		return "unknown"
	}
	if loc != nil {
		ts = ts.In(loc)
	}
	return ts.Format("Jan 2 15:04:05")
}

func formatPipelineAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return d.String()
	}
	return d.Round(time.Second).String()
}

func bufferTailFromReader(logReader io.Reader, trunc string, buffsize, linesize int) []byte {
	tail := linebuffer.New(buffsize, linesize, trunc)
	scanner := bufio.NewScanner(logReader)
	for scanner.Scan() {
		tail.WriteLine(scanner.Text())
	}
	tail.Close()
	tailReader, _ := tail.Reader()
	buff, _ := io.ReadAll(tailReader)
	return buff
}

func (w *worker) liveLogBuffer() *pipelineLiveLogger {
	w.Lock()
	defer w.Unlock()
	return w.liveLogger
}

func (w *worker) liveLogSnapshot() string {
	logger := w.liveLogBuffer()
	if logger == nil {
		return ""
	}
	data, _ := io.ReadAll(logger.Snapshot())
	return strings.TrimSpace(string(data))
}

func (w *worker) liveLogExcerpt() string {
	logger := w.liveLogBuffer()
	if logger == nil {
		return ""
	}
	data := bufferTailFromReader(logger.Snapshot(), " ...", tailBody, tailLine)
	return strings.TrimSpace(string(data))
}

func (w *worker) startPipelineWatchdog() {
	w.Lock()
	timeOuts := w.timeOuts
	startedAt := w.startedAt
	if !timeOuts.any() {
		w.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.watchdogCancel = cancel
	w.Unlock()

	schedule := func(delay time.Duration, fn func()) {
		if delay <= 0 {
			go fn()
			return
		}
		go func() {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				fn()
			}
		}()
	}

	if timeOuts.Warn > 0 {
		schedule(time.Until(startedAt.Add(timeOuts.Warn)), w.emitPipelineTimeOutWarn)
	}
	if timeOuts.Kill > 0 {
		schedule(time.Until(startedAt.Add(timeOuts.Kill)), w.emitPipelineTimeOutKill)
	}
}

func (w *worker) stopPipelineWatchdog() {
	w.Lock()
	cancel := w.watchdogCancel
	w.watchdogCancel = nil
	w.Unlock()
	if cancel != nil {
		cancel()
	}
}

type timeoutInterruptResult struct {
	killed bool
	pid    int
	err    error
	manual bool
}

func interruptPipelineForTimeOut(worker *worker) timeoutInterruptResult {
	var pid int
	var activeTaskTID int
	var rpcCancel context.CancelFunc
	worker.Lock()
	if worker.osCmd != nil {
		pid = worker.osCmd.Process.Pid
	}
	activeTaskTID = worker.activeTaskTID
	rpcCancel = worker.rpcCancel
	worker.Unlock()

	if rpcCancel != nil {
		rpcCancel()
	}
	_ = interruptReplyWaitersForTask(activeTaskTID)

	if pid == 0 {
		return timeoutInterruptResult{manual: true}
	}
	raiseThreadPriv(fmt.Sprintf("timeout kill for pipeline %d", worker.id))
	if err := unix.Kill(-pid, unix.SIGKILL); err != nil && err != unix.ESRCH {
		return timeoutInterruptResult{pid: pid, err: err}
	}
	return timeoutInterruptResult{killed: true, pid: pid}
}

func (w *worker) formatPipelineAlert(title string, extra ...string) string {
	w.Lock()
	startedAt := w.startedAt
	timeZone := w.timeZone
	pipeName := w.pipeName
	taskName := w.taskName
	taskType := w.taskType
	command := w.plugCommand
	args := append([]string(nil), w.taskArgs...)
	wid := w.id
	w.Unlock()

	lines := []string{title}
	lines = append(lines, fmt.Sprintf("WID: `%d`", wid))
	lines = append(lines, fmt.Sprintf("Pipeline: `%s`", pipeName))
	if taskName != "" {
		current := taskName
		if taskType == "plugin" && command != "" {
			current = fmt.Sprintf("%s/%s", taskName, command)
		}
		if len(args) > 0 {
			current += " " + strings.Join(args, " ")
		}
		lines = append(lines, fmt.Sprintf("Current task: `%s`", current))
	}
	lines = append(lines, fmt.Sprintf("Started: `%s`", formatPipelineClock(startedAt, timeZone)))
	lines = append(lines, fmt.Sprintf("Age: `%s`", formatPipelineAge(time.Since(startedAt))))
	lines = append(lines, extra...)
	excerpt := w.liveLogExcerpt()
	if excerpt != "" {
		lines = append(lines, "Recent log:")
		lines = append(lines, "```")
		lines = append(lines, excerpt)
		lines = append(lines, "```")
	}
	return strings.Join(lines, "\n")
}

func (w *worker) sendPipelineAlert(message string) {
	w.Lock()
	channel := strings.TrimSpace(w.operatorChannel)
	w.Unlock()
	if channel == "" {
		Log(robot.Warn, "No operator channel configured for pipeline '%s'; skipping admin alert", w.pipeName)
		return
	}
	r := w.makeRobot()
	if ret := r.MessageFormat(robot.BasicMarkdown).SendChannelMessage(channel, message); ret != robot.Ok {
		Log(robot.Warn, "Unable to send pipeline alert for '%s' to channel '%s': %s", w.pipeName, channel, ret)
	}
}

func (w *worker) emitPipelineTimeOutWarn() {
	w.Lock()
	if !w.active || w.timeOutWarnSent {
		w.Unlock()
		return
	}
	w.timeOutWarnSent = true
	logger := w.liveLogger
	w.Unlock()
	if logger != nil {
		logger.Line("*** timeout - warn threshold reached")
	}
	w.sendPipelineAlert(w.formatPipelineAlert("Pipeline timeout warning", "The configured warn threshold has been reached."))
}

func (w *worker) emitPipelineTimeOutKill() {
	w.Lock()
	if !w.active || w.timeOutKillSent {
		w.Unlock()
		return
	}
	w.timeOutKillSent = true
	logger := w.liveLogger
	w.Unlock()
	if logger != nil {
		logger.Line("*** timeout - kill threshold reached")
	}
	result := interruptPipelineForTimeOut(w)
	w.Lock()
	w.timeOutKillManual = result.manual
	w.Unlock()
	if result.err != nil {
		w.sendPipelineAlert(w.formatPipelineAlert(
			"Pipeline timeout kill failed",
			fmt.Sprintf("The engine tried to kill the active process for this pipeline and got: `%v`", result.err),
		))
		return
	}
	if result.manual {
		w.sendPipelineAlert(w.formatPipelineAlert(
			"Pipeline timeout kill threshold reached",
			"This pipeline is currently running in-process Go work, so manual intervention is required.",
		))
		return
	}
	w.sendPipelineAlert(w.formatPipelineAlert(
		"Pipeline timeout kill threshold reached",
		fmt.Sprintf("The engine sent a kill signal to process `%d`.", result.pid),
	))
}

func (w *worker) emitPipelineFailureAlert(ret robot.TaskRetVal, errString string) {
	w.Lock()
	if !w.executedPrimaryTask || w.timeOutKillSent {
		w.Unlock()
		return
	}
	w.Unlock()
	title := fmt.Sprintf("Pipeline failure: exit code %d (%s)", ret, ret)
	extra := []string{}
	if strings.TrimSpace(errString) != "" {
		extra = append(extra, fmt.Sprintf("Failure detail: `%s`", strings.TrimSpace(errString)))
	}
	w.sendPipelineAlert(w.formatPipelineAlert(title, extra...))
}
