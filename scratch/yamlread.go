package main

import (
	//	"encoding/json"
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
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
	Conf       yaml.MapSlice
	Role       string
	Matcher    string
}

func main() {
	var c confmain
	b, _ := ioutil.ReadFile("user.yaml")
	yaml.Unmarshal(b, &c)
	pbc, _ := yaml.Marshal(c.Conf)
	fmt.Printf("%s", string(pbc))
	var pc pconf
	yaml.Unmarshal(pbc, &pc)
	fmt.Printf("We got: %v\n", pc)
}
