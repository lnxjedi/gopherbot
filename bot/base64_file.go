package bot

import (
	"encoding/base64"
	"io"
	"os"

	"github.com/emersion/go-textwrapper"
)

var base64header = "#GOPHERBOT-ENCRYPTED-BASE64\n"

// base64_file.go - read and write #GOPHERBOT-ENCRYPTED-BASE64 files

// WriteBase64File writes a byte slice to a #GOPHERBOT-ENCRYPTED-BASE64
// file
func WriteBase64File(filename string, b *[]byte) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return WriteBase64(f, b)
}

// WriteBase64 writes out a #GOPHERBOT-ENCRYPTED-BASE64 file
func WriteBase64(out io.Writer, b *[]byte) error {
	out.Write([]byte(base64header))
	w := textwrapper.New(out, "\n", 77)
	encoder := base64.NewEncoder(base64.StdEncoding, w)
	encoder.Write(*b)
	encoder.Close()
	w.Write([]byte("\n"))
	return nil
}

// func ReadBase64File(filename string) (*[]byte, error) {

// }
