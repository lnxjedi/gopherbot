package bot

func getProtocolChannelByName(maps *userChanMaps, protocol, channel string) (*ChannelInfo, bool) {
	if maps == nil {
		return nil, false
	}
	p := normalizeProtocolName(protocol)
	if p != "" {
		if pm, ok := maps.channelProto[p]; ok {
			if ci, ok := pm[channel]; ok {
				return ci, true
			}
		}
	}
	ci, ok := maps.channel[channel]
	return ci, ok
}

func getProtocolChannelByID(maps *userChanMaps, protocol, channelID string) (*ChannelInfo, bool) {
	if maps == nil {
		return nil, false
	}
	p := normalizeProtocolName(protocol)
	if p != "" {
		if pm, ok := maps.channelIDProto[p]; ok {
			if ci, ok := pm[channelID]; ok {
				return ci, true
			}
		}
	}
	ci, ok := maps.channelID[channelID]
	return ci, ok
}
