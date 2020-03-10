package bot

// memhistory provides a trivial history provider that only keeps
// active (un-Finalize()'d) histories in 64k buffers

import "github.com/lnxjedi/robot"

const logSize = 65536

type memlogentry struct {
	tag string
	idx int
}

type memlog struct {
	log []byte
	start, end int
}

var histLogs = struct {
	logs map[memlogentry]
}

func mhprovider(r robot.Handler) robot.HistoryProvider {

}