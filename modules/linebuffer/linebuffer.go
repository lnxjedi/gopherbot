// Package linebuffer provides a line-oriented circular buffer with truncation and paging.
// It stores UTF-8 text in lines up to a maximum length, truncates oversize lines with
// a truncation string, and discards oldest lines when it runs out of space.
//
// Typical usage:
//
//	lb := linebuffer.New(64*1024, 2048, "...(truncated)\n")
//	lb.WriteLine("A short line")
//	lb.WriteLine("A very very long line that needs truncation ...")
//	lb.Close()
//	r := lb.Reader()
//	// read from r in chunks or process as needed
//
// Constraints:
//   - linesize <= buffsize to ensure every truncated line fits into the buffer.
//   - Once Close() is called, no more writes can occur.
//   - The Reader() provides a snapshot of the stored lines at close time.
//   - On overflow, oldest lines are discarded.
package linebuffer

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
)

type Buffer struct {
	buf      []byte
	size     int
	linesize int
	trunc    string
	start    int
	length   int
	closed   bool
	mu       sync.Mutex
}

// reader is an io.Reader that reads from a snapshot of the Buffer’s data.
type reader struct {
	buf      *Buffer
	position int
	snapshot []byte
}

// New creates a new line buffer.
// buffsize: maximum size of the buffer in bytes
// linesize: maximum size of a single line (including truncation string), must be <= buffsize
// truncstr: a string appended to truncated lines to indicate truncation, must end with "\n"
func New(buffsize, linesize int, truncstr string) *Buffer {
	if !strings.HasSuffix(truncstr, "\n") {
		truncstr += "\n"
	}
	if linesize > buffsize {
		panic("linesize must be <= buffsize")
	}
	return &Buffer{
		buf:      make([]byte, buffsize),
		size:     buffsize,
		linesize: linesize,
		trunc:    truncstr,
	}
}

// Close marks the buffer as closed. No more lines can be written after Close.
func (b *Buffer) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}

// WriteLine writes a single line to the buffer.
// If it doesn't end with a newline, one is appended.
// If it's too long, it is truncated, and truncstr is appended.
func (b *Buffer) WriteLine(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}

	line = b.normalizeLine(line)
	line = b.truncateLine(line)
	lsize := len(line)

	b.ensureSpace(lsize)
	b.writeBytes(line)
}

// Reader returns an io.Reader after the buffer is closed.
// If the buffer is not closed, returns an error.
// The returned reader reads a snapshot of the current buffer contents.
func (b *Buffer) Reader() (io.Reader, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		return nil, errors.New("buffer not closed")
	}

	// Create a snapshot of current data in correct order.
	snap := make([]byte, b.length)
	if b.length == 0 {
		return bytes.NewReader(snap), nil
	}

	end := (b.start + b.length) % b.size
	if b.start < end {
		copy(snap, b.buf[b.start:end])
	} else {
		// Wrapped around
		n1 := copy(snap, b.buf[b.start:])
		copy(snap[n1:], b.buf[:end])
	}

	return bytes.NewReader(snap), nil
}

// Snapshot returns a reader for the current contents of the buffer
// without requiring it to be closed. The returned reader is a snapshot
// and won't be affected by future writes.
func (b *Buffer) Snapshot() io.Reader {
	b.mu.Lock()
	defer b.mu.Unlock()

	snap := make([]byte, b.length)
	if b.length > 0 {
		end := (b.start + b.length) % b.size
		if b.start < end {
			copy(snap, b.buf[b.start:end])
		} else {
			n := copy(snap, b.buf[b.start:])
			copy(snap[n:], b.buf[:end])
		}
	}

	return bytes.NewReader(snap)
}

// normalizeLine ensures the line ends with a newline.
func (b *Buffer) normalizeLine(line string) string {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	return line
}

// truncateLine truncates the line if it's longer than linesize,
// appending the truncation string.
func (b *Buffer) truncateLine(line string) string {
	if len(line) > b.linesize {
		truncLen := b.linesize - len(b.trunc)
		if truncLen < 0 {
			// If truncation string itself is too large, panic or handle error
			panic("truncation string too large for linesize")
		}
		line = line[:truncLen] + b.trunc
	}
	return line
}

// ensureSpace makes room for a line of length lsize.
// If there isn't enough space, remove oldest lines until there is.
func (b *Buffer) ensureSpace(lsize int) {
	// If there's already enough room, do nothing.
	if b.length+lsize <= b.size {
		return
	}

	// Need to remove lines until we have room.
	// Because we store line-based data, we must discard line by line.
	// We'll discard whole lines from the start until we have enough space.
	spaceNeeded := (b.length + lsize) - b.size
	b.discardLines(spaceNeeded)
}

// discardLines removes at least "spaceNeeded" bytes from the oldest data.
func (b *Buffer) discardLines(spaceNeeded int) {
	// We'll remove one line at a time by scanning for a newline.
	// Continue until we free enough space.
	for spaceNeeded > 0 && b.length > 0 {
		// Find the next newline from start to find the first line boundary.
		endLine := b.indexOfNewline()
		if endLine == -1 {
			// If no newline found, that means something is wrong or only partial line.
			// In theory, we always store complete lines.
			// Just clear everything if that happens.
			b.start = 0
			b.length = 0
			break
		}
		lineLen := endLine + 1 // include newline char
		b.start = (b.start + lineLen) % b.size
		b.length -= lineLen
		spaceNeeded -= lineLen
	}
}

// indexOfNewline finds the next newline in the buffer starting at b.start.
// Returns the index relative to b.start, or -1 if none found.
func (b *Buffer) indexOfNewline() int {
	if b.length == 0 {
		return -1
	}
	// The buffered data is in [start:start+length] modulo size
	end := (b.start + b.length) % b.size
	if b.start < end {
		return bytes.IndexByte(b.buf[b.start:end], '\n')
	}

	// Wrapped around - search in two parts
	part1 := bytes.IndexByte(b.buf[b.start:], '\n')
	if part1 != -1 {
		return part1
	}
	part2 := bytes.IndexByte(b.buf[:end], '\n')
	if part2 != -1 {
		// part2 + length of the first segment
		return (b.size - b.start) + part2
	}

	return -1
}

// writeBytes writes a line's bytes into the buffer and updates length accordingly.
func (b *Buffer) writeBytes(line string) {
	lsize := len(line)
	// end position
	end := (b.start + b.length) % b.size

	// Copy line into buffer
	n := copy(b.buf[end:], line)
	if n < lsize {
		// wrapped
		copy(b.buf, line[n:])
	}

	b.length += lsize
}
