package bot

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lnxjedi/robot"
)

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
	decFlags.BoolVar(&encodeBinary, "binary", false, "")
	decFlags.BoolVar(&encodeBinary, "b", false, "")
	decFlags.Usage = func() {
		fmt.Println("Usage: gopherbot decrypt [options] [string to decrypt]\n\nOptions:")
		decFlags.PrintDefaults()
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
	case "init":
		if len(cliArgs) < 2 {
			fmt.Println("Usage: gopherbot init <protocol>")
			return
		}
		if _, err := os.Stat("answerfile.txt"); err == nil {
			fmt.Println("Not over-writing existing 'answerfile.txt'")
			return
		}
		ansFile := filepath.Join(installPath, "resources", "answerfiles", cliArgs[1]+".txt")
		if _, err := os.Stat(ansFile); err != nil {
			fmt.Printf("Protocol answerfile template not found: %s\n", ansFile)
			return
		}
		var ansBytes []byte
		var err error
		if ansBytes, err = ioutil.ReadFile(ansFile); err != nil {
			fmt.Printf("Reading '%s': %v", ansFile, err)
			return
		}
		if err = ioutil.WriteFile("answerfile.txt", ansBytes, 0600); err != nil {
			fmt.Printf("Writing 'answerfile.txt': %v", err)
			return
		}
		if _, err := os.Stat("gopherbot"); err == nil {
			fmt.Println("Edit 'answerfile.txt' and re-run gopherbot with no arguments to generate your robot.")
		} else {
			exeFile := filepath.Join(installPath, "gopherbot")
			err := os.Symlink(exeFile, "gopherbot")
			if err != nil {
				fmt.Println("Unable to create symlink for 'gopherbot'")
				fmt.Println("Edit 'answerfile.txt' and re-run gopherbot with no arguments to generate your robot.")
			} else {
				fmt.Println("Edit 'answerfile.txt' and run './gopherbot' with no arguments to generate your robot.")
			}
		}
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
			WriteBase64(os.Stdout, &ct)
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
		var ct *[]byte
		var err error
		if file == "-" {
			ct, err = ReadBinary(os.Stdin)
		} else {
			ct, err = ReadBinaryFile(file)
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		pt, err := decrypt(*ct, cryptKey.key)
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
	if len(list) > 0 {
		for _, memory := range list {
			fmt.Println(memory)
		}
		return
	}
	fmt.Println("No memories found")
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
