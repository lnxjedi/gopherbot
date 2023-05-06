package bot

import (
	"bufio"
	"encoding/base64"
	"io"
	"os"

	"github.com/emersion/go-textwrapper"
)

var base64header = "#GOPHERBOT-BASE64-DATA\n"

// base64_file.go - read and write #GOPHERBOT-BASE64-DATA files

// WriteBase64File writes a byte slice to a #GOPHERBOT-BASE64-DATA
// file
func WriteBase64File(filename string, b *[]byte) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	return WriteBase64(f, b)
}

// WriteBase64 writes out a #GOPHERBOT-BASE64-DATA file
func WriteBase64(out io.Writer, b *[]byte) error {
	out.Write([]byte(base64header))
	w := textwrapper.New(out, "\n", 77)
	encoder := base64.NewEncoder(base64.StdEncoding, w)
	encoder.Write(*b)
	encoder.Close()
	w.Write([]byte("\n"))
	return nil
}

// ReadBinaryFile reads a binary file, detecting and decoding base64
func ReadBinaryFile(filename string) (*[]byte, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return ReadBinary(in)
}

// ReadBinary reads a binary reader, detecting and decoding base64
func ReadBinary(in io.Reader) (*[]byte, error) {
	br := bufio.NewReader(in)
	header, err := br.Peek(len(base64header))
	if err == nil {
		if string(header) == base64header {
			br.Discard(len(base64header))
			decoder := base64.NewDecoder(base64.StdEncoding, br)
			bytes, err := io.ReadAll(decoder)
			if err != nil {
				return nil, err
			}
			return &bytes, nil
		}
	}
	bytes, err := io.ReadAll(br)
	if err == nil {
		return &bytes, nil
	}
	return nil, err
}
