package bot

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/lnxjedi/gopherbot/robot"
)

var base64header = "#GOPHERBOT-ENCRYPTED-BASE64\n"

func processCLI(usage string) {
	cliArgs := flag.Args()
	command := cliArgs[0]

	var fileName string
	var encodeBinary bool
	var encodeBase64 bool

	encFlags := flag.NewFlagSet("encrypt", flag.ExitOnError)
	encFlags.StringVar(&fileName, "file", "", "file to encrypt (or - for stdout)")
	encFlags.StringVar(&fileName, "f", "", "")
	encFlags.BoolVar(&encodeBinary, "binary", false, "binary dump (defauts to base64 encoded)")
	encFlags.BoolVar(&encodeBinary, "b", false, "")
	encFlags.Usage = func() {
		fmt.Println("Usage: gopherbot encrypt [options] [string to encrypt]\n\nOptions:")
		encFlags.PrintDefaults()
	}

	decFlags := flag.NewFlagSet("decrypt", flag.ExitOnError)
	decFlags.StringVar(&fileName, "file", "", "file to decrypt (or - for stdin)")
	decFlags.StringVar(&fileName, "f", "", "")
	decFlags.Usage = func() {
		fmt.Println("Usage: gopherbot decrypt [options] [string to decrypt]\n\nOptions:")
		encFlags.PrintDefaults()
	}

	fetchFlags := flag.NewFlagSet("fetch", flag.ExitOnError)
	fetchFlags.BoolVar(&encodeBase64, "base64", false, "encode memory as base64")
	fetchFlags.BoolVar(&encodeBase64, "b", false, "")
	fetchFlags.Usage = func() {
		fmt.Println("Usage: gopherbot fetch [options] <memory to fetch>\n\nOptions:")
		fetchFlags.PrintDefaults()
	}

	switch command {
	case "encrypt":
		encFlags.Parse(cliArgs[1:])
		if len(fileName) == 0 && len(encFlags.Args()) != 1 {
			encFlags.Usage()
			return
		}
		cliEncrypt(encFlags.Arg(0), fileName, encodeBinary)
	case "decrypt":
		decFlags.Parse(cliArgs[1:])
		if len(fileName) == 0 && len(decFlags.Args()) != 1 {
			decFlags.Usage()
			return
		}
		cliDecrypt(decFlags.Arg(0), fileName)
	case "fetch":
		fetchFlags.Parse(cliArgs[1:])
		if len(fetchFlags.Args()) == 0 || len(fetchFlags.Arg(0)) == 0 {
			fetchFlags.Usage()
			return
		}
		cliFetch(fetchFlags.Arg(0), encodeBase64)
	case "store":
		if len(cliArgs) < 2 {
			fmt.Println("Usage: gopherbot store <key> [filename]")
			return
		}
		file := "-"
		if len(cliArgs) == 3 {
			file = cliArgs[2]
		}
		cliStore(cliArgs[1], file)
	case "list":
		cliList()
	case "delete":
		if len(cliArgs) != 2 {
			fmt.Println("Usage: gopherbot delete <key>")
			return
		}
		cliDelete(cliArgs[1])
	default:
		fmt.Println(usage)
		flag.PrintDefaults()
	}
}

func cliEncrypt(item, file string, binary bool) {
	if !cryptKey.initialized {
		fmt.Println("Encryption not initialized")
		os.Exit(1)
	}
	if len(file) > 0 {
		var fc []byte
		var err error
		if file == "-" {
			fc, err = ioutil.ReadAll(os.Stdin)
		} else {
			fc, err = ioutil.ReadFile(file)
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		ct, err := encrypt(fc, cryptKey.key)
		if binary {
			os.Stdout.Write(ct)
		} else {
			os.Stdout.Write([]byte(base64header))
			encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
			encoder.Write(ct)
			encoder.Close()
			os.Stdout.Write([]byte("\n"))
		}
		return
	}
	if len(item) > 0 {
		ct, err := encrypt([]byte(item), cryptKey.key)
		if err != nil {
			fmt.Printf("Error encrypting: %v\n", err)
			os.Exit(1)
		}
		if binary {
			os.Stdout.Write(ct)
		} else {
			fmt.Println(base64.StdEncoding.EncodeToString(ct))
		}
		return
	}
	os.Stderr.Write([]byte("Ingoring zero-length item\n"))
	os.Exit(1)
}

func cliDecrypt(item, file string) {
	if !cryptKey.initialized {
		fmt.Println("Encryption not initialized")
		os.Exit(1)
	}
	if len(file) > 0 {
		var fc, ct []byte
		var err error
		if file == "-" {
			fc, err = ioutil.ReadAll(os.Stdin)
		} else {
			fc, err = ioutil.ReadFile(file)
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		if string(fc[0:len(base64header)]) == base64header {
			decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(fc[len(base64header):]))
			ct, err = ioutil.ReadAll(decoder)
		} else {
			ct = fc
		}
		pt, err := decrypt(ct, cryptKey.key)
		if err != nil {
			fmt.Printf("Error decrypting: %v\n", err)
		}
		os.Stdout.Write(pt)
		return
	}
	if len(item) > 0 {
		eb, err := base64.StdEncoding.DecodeString(item)
		if err != nil {
			fmt.Printf("Decoding base64: %v\n", err)
			os.Exit(1)
		}
		value, err := decrypt(eb, cryptKey.key)
		if err != nil {
			fmt.Printf("Error decrypting: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(value))
		return
	}
	os.Stderr.Write([]byte("Ingoring zero-length item\n"))
	os.Exit(1)
}

func cliFetch(item string, b64 bool) {
	_, datum, exists, ret := getDatum(item, false)
	if ret != robot.Ok {
		fmt.Printf("Retrieving datum: %v\n", ret)
		os.Exit(1)
	}
	if !exists {
		fmt.Println("Item not found")
		os.Exit(1)
	}
	if b64 {
		encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
		encoder.Write(*datum)
		os.Stdout.Write([]byte("\n"))
		return
	}
	os.Stdout.Write(*datum)
	os.Stdout.Write([]byte("\n"))
}

func cliStore(key, file string) {
	var fc []byte
	var err error
	if file == "-" {
		fc, err = ioutil.ReadAll(os.Stdin)
	} else {
		fc, err = ioutil.ReadFile(file)
	}
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	tok, _, _, ret := checkout(key, true)
	if ret != robot.Ok {
		fmt.Printf("Getting token: %s\n", ret)
		return
	}
	ret = update(key, tok, &fc)
	if ret != robot.Ok {
		fmt.Printf("Storing datum: %s\n", ret)
		return
	}
	fmt.Println("Stored")
}

func cliList() {
	brain := interfaces.brain
	list, err := brain.List()
	if err != nil {
		fmt.Printf("Listing memories: %v\n", err)
		return
	}
	for _, memory := range list {
		fmt.Println(memory)
	}
}

func cliDelete(key string) {
	brain := interfaces.brain
	err := brain.Delete(key)
	if err != nil {
		fmt.Printf("Deleting memory: %v\n", err)
		return
	}
	fmt.Println("Deleted")
}
