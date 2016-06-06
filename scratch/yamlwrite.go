package main

import (
	//	"encoding/json"
	"fmt"
	//	"io/ioutil"
	"github.com/go-yaml/yaml"
)

type pet struct {
	Name, Breed string
}

type pconf struct {
	Job  string
	Pets []pet
}

type confmain struct {
	Name, Size string
	Age        int
	Conf       pconf
	Role       string
	Matcher    string
}

func main() {
	c := confmain{
		Name: "Bill",
		Size: "large",
		Age:  15,
		Conf: pconf{
			Job: "Janitor",
			Pets: []pet{
				{"Jim", "Koala"},
				{"Frank", "Bear"},
			},
		},
		Role:    "Worker bee",
		Matcher: `(?i:[\w-. ]+)`,
	}
	b, _ := yaml.Marshal(c)
	fmt.Printf("%s", string(b))
}
