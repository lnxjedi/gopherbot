package bot

import (
	"fmt"
	"math"
	"testing"
)

type logStateSnapshot struct {
	buffer    []string
	buffLine  int
	pageLines int
	buffPages int
	totLines  int
}

func snapshotLogState() logStateSnapshot {
	botLogger.Lock()
	defer botLogger.Unlock()
	buf := make([]string, len(botLogger.buffer))
	copy(buf, botLogger.buffer)
	return logStateSnapshot{
		buffer:    buf,
		buffLine:  botLogger.buffLine,
		pageLines: botLogger.pageLines,
		buffPages: botLogger.buffPages,
		totLines:  botLogger.totLines,
	}
}

func restoreLogState(s logStateSnapshot) {
	botLogger.Lock()
	defer botLogger.Unlock()
	copy(botLogger.buffer, s.buffer)
	botLogger.buffLine = s.buffLine
	botLogger.pageLines = s.pageLines
	botLogger.buffPages = s.buffPages
	botLogger.totLines = s.totLines
}

func TestLogPageBasicPagingAndWrap(t *testing.T) {
	s := snapshotLogState()
	defer restoreLogState(s)

	botLogger.Lock()
	for i := range botLogger.buffer {
		botLogger.buffer[i] = ""
	}
	botLogger.pageLines = 2
	botLogger.buffLine = 5
	botLogger.totLines = 5
	botLogger.buffer[0] = "L1"
	botLogger.buffer[1] = "L2"
	botLogger.buffer[2] = "L3"
	botLogger.buffer[3] = "L4"
	botLogger.buffer[4] = "L5"
	botLogger.Unlock()

	page0, wrapped0 := logPage(0)
	if wrapped0 {
		t.Fatal("page 0 should not wrap")
	}
	if fmt.Sprint(page0) != "[L4 L5]" {
		t.Fatalf("page0=%v, want [L4 L5]", page0)
	}

	page1, wrapped1 := logPage(1)
	if wrapped1 {
		t.Fatal("page 1 should not wrap")
	}
	if fmt.Sprint(page1) != "[L2 L3]" {
		t.Fatalf("page1=%v, want [L2 L3]", page1)
	}

	page2, wrapped2 := logPage(2)
	if wrapped2 {
		t.Fatal("page 2 should not wrap")
	}
	if fmt.Sprint(page2) != "[L1]" {
		t.Fatalf("page2=%v, want [L1]", page2)
	}

	page3, wrapped3 := logPage(3)
	if !wrapped3 {
		t.Fatal("page 3 should wrap")
	}
	if fmt.Sprint(page3) != "[L4 L5]" {
		t.Fatalf("page3=%v, want [L4 L5]", page3)
	}
}

func TestLogPageWrappedBufferOrder(t *testing.T) {
	s := snapshotLogState()
	defer restoreLogState(s)

	botLogger.Lock()
	for i := range botLogger.buffer {
		botLogger.buffer[i] = ""
	}
	botLogger.pageLines = 20
	botLogger.totLines = buffLines + 10
	botLogger.buffLine = 10
	oldest := (botLogger.buffLine - buffLines + buffLines) % buffLines
	firstSeq := botLogger.totLines - buffLines + 1
	for i := 0; i < buffLines; i++ {
		idx := (oldest + i) % buffLines
		botLogger.buffer[idx] = fmt.Sprintf("L%03d", firstSeq+i)
	}
	botLogger.Unlock()

	page0, wrapped := logPage(0)
	if wrapped {
		t.Fatal("latest page should not wrap")
	}
	if len(page0) != 20 {
		t.Fatalf("len(page0)=%d, want 20", len(page0))
	}
	if page0[0] != "L491" || page0[len(page0)-1] != "L510" {
		t.Fatalf("unexpected latest page bounds: first=%q last=%q", page0[0], page0[len(page0)-1])
	}
}

func TestSetLogPageLinesBounds(t *testing.T) {
	s := snapshotLogState()
	defer restoreLogState(s)

	if got := setLogPageLines(0); got != 1 {
		t.Fatalf("setLogPageLines(0)=%d, want 1", got)
	}
	botLogger.Lock()
	if botLogger.pageLines != 1 {
		t.Fatalf("pageLines=%d, want 1", botLogger.pageLines)
	}
	expectedPages := int(math.Ceil(float64(buffLines) / float64(botLogger.pageLines)))
	if botLogger.buffPages != expectedPages {
		t.Fatalf("buffPages=%d, want %d", botLogger.buffPages, expectedPages)
	}
	botLogger.Unlock()

	if got := setLogPageLines(maxLines + 10); got != maxLines {
		t.Fatalf("setLogPageLines(max+10)=%d, want %d", got, maxLines)
	}
}
