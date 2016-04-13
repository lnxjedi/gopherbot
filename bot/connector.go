package gobot

// type Channel is a name to id mapping struct
type Channel struct {
	id   string // protocol internal ID, may be meaningless to humans
	name string // human-friendly name
}

// type Connector is an interface that all protocols must implement
type Connector interface {
	//	GetChannelID(name string) string
	//	GetChannelName(id string) string
	SendChannelMessage(chanid string, msg string)
}
