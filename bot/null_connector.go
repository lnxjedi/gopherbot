package bot

import (
	"log"

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

func (nc nullConnector) SendProtocolChannelMessage(ch string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	return robot.Ok
}

func (nc nullConnector) SendProtocolUserChannelMessage(uid, uname, ch, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	return robot.Ok
}

func (nc nullConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	return robot.Ok
}

func (nc nullConnector) SetUserMap(map[string]string) {
	return
}
