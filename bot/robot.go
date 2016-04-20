package bot

/* robot.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

func (r Robot) SendChannelMessage(ch, msg string) {
	f := setFormat(r.Format)
	r.SendProtocolChannelMessage(ch, msg, f)
}

func (r Robot) SendUserChannelMessage(u, ch, msg string) {
	f := setFormat(r.Format)
	r.SendProtocolUserChannelMessage(u, ch, msg, f)
}

func (r Robot) SendUserMessage(u, msg string) {
	f := setFormat(r.Format)
	r.SendProtocolUserMessage(u, msg, f)
}
