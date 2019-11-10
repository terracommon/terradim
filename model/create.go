package model

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/derekparker/trie"
	"gopkg.in/yaml.v2"
)

// NodeConfig comment
type NodeConfig struct {
	A string
	B struct {
		C int   `yaml:"c"`
		D []int `yaml:",flow"`
	}
}

// NodeMeta comment
type NodeMeta struct {
	Size  int64
	IsDir bool
	//isEnum bool
	Config NodeConfig
}

// Create trie model
func Create(dirpath string) *trie.Trie {
	model := trie.New()

	err := filepath.Walk(dirpath,
		func(curpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			var config NodeConfig
			_, file := filepath.Split(curpath)

			if file == "dim1.yml" {
				config, err = loadConfig(curpath)
				check(err)

			}
			meta := NodeMeta{Size: info.Size(), IsDir: info.IsDir(), Config: config}
			fmt.Println(curpath, meta)
			model.Add(curpath, meta)
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return model
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func loadConfig(filepath string) (NodeConfig, error) {
	var config NodeConfig

	filedata, err := ioutil.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	yaml.Unmarshal(filedata, &config)
	fmt.Printf("Yaml values:\n%v\n", config)
	return config, nil
}
