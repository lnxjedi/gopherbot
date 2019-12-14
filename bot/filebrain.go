package bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

var brainPath string
var fbhandler robot.Handler

type fbConfig struct {
	BrainDirectory string // path to brain files
	Encode         bool   // whether to base64 encode memories, default true
}

var fb fbConfig

func (fb *fbConfig) Store(k string, b *[]byte) error {
	k = strings.Replace(k, `/`, ":", -1)
	k = strings.Replace(k, `\`, ":", -1)
	datumPath := filepath.Join(brainPath, k)
	if fb.Encode {
		if err := WriteBase64File(datumPath, b); err != nil {
			return fmt.Errorf("writing datum '%s': %v", datumPath, err)
		}
		return nil
	}
	if err := ioutil.WriteFile(datumPath, *b, 0600); err != nil {
		return fmt.Errorf("Writing datum '%s': %v", datumPath, err)
	}
	return nil
}

func (fb *fbConfig) Retrieve(k string) (*[]byte, bool, error) {
	k = strings.Replace(k, `/`, ":", -1)
	k = strings.Replace(k, `\`, ":", -1)
	datumPath := filepath.Join(brainPath, k)
	if _, err := os.Stat(datumPath); err == nil {
		datum, err := ReadBinaryFile(datumPath)
		if err != nil {
			err = fmt.Errorf("Error reading file \"%s\": %v", datumPath, err)
			fbhandler.Log(robot.Error, err.Error())
			return nil, false, err
		}
		return datum, true, nil
	}
	// Memory doesn't exist yet
	return nil, false, nil
}

func (fb *fbConfig) List() ([]string, error) {
	d, err := os.Open(brainPath)
	if err != nil {
		return []string{}, err
	}
	keys, err := d.Readdirnames(0)
	if err != nil {
		return []string{}, err
	}
	return keys, nil
}

func (fb *fbConfig) Delete(key string) (err error) {
	err = os.Remove(filepath.Join(brainPath, key))
	return
}

// The file brain doesn't need the logger, but other brains might
func fbprovider(r robot.Handler) robot.SimpleBrain {
	fbhandler = r
	fbhandler.GetBrainConfig(&fb)
	if len(fb.BrainDirectory) == 0 {
		fbhandler.Log(robot.Fatal, "BrainConfig missing value for BrainDirectory required by 'file' brain")
	}
	brainPath = fb.BrainDirectory
	if err := r.GetDirectory(brainPath); err != nil {
		fbhandler.Log(robot.Fatal, "Getting brain directory \"%s\": %v", brainPath, err)
	}
	fbhandler.Log(robot.Info, "Initialized file-backed brain with memories directory: '%s'", brainPath)
	return &fb
}

func init() {
	RegisterSimpleBrain("file", fbprovider)
}
