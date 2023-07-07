package bot

import (
	"fmt"
	"log"
	"os"

	"github.com/lnxjedi/gopherbot/robot"
)

type nullConnector struct{}

func nullStart(robot.Handler, *log.Logger) robot.Connector {
	nc := nullConnector{}
	return nc
}

func init() {
	RegisterConnector("nullconn", nullStart)
}

func (nc nullConnector) GetProtocolUserAttribute(u, a string) (value string, ret robot.RetVal) {
	return
}

func (nc nullConnector) JoinChannel(c string) robot.RetVal {
	return robot.Ok
}

func (nc nullConnector) MessageHeard(u, c string) {
	return
}

func (nc nullConnector) Run(stop <-chan struct{}) {
	<-stop
}

func (nc nullConnector) FormatHelp(input string) string {
	return input
}

func (nc nullConnector) SendProtocolChannelThreadMessage(ch string, thr, msg string, f robot.MessageFormat, dummyMsgObject interface{}) (ret robot.RetVal) {
	return nc.sendMessage(msg, f)
}

func (nc nullConnector) SendProtocolUserChannelThreadMessage(uid, uname, ch, thr, msg string, f robot.MessageFormat, dummyMsgObject interface{}) (ret robot.RetVal) {
	return nc.sendMessage(msg, f)
}

func (nc nullConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat, dummyMsgObject interface{}) (ret robot.RetVal) {
	return nc.sendMessage(msg, f)
}

func (nc nullConnector) SetUserMap(map[string]string) {
	return
}

func (nc nullConnector) sendMessage(msg string, f robot.MessageFormat) (ret robot.RetVal) {
	output := fmt.Sprintf("null connector: %s\n", msg)
	if f != robot.Fixed {
		output = Wrap(output, 80)
		os.Stdout.Write([]byte(output)[0 : len(output)-1])
	} else {
		os.Stdout.Write([]byte(output))
	}
	return robot.Ok
}
