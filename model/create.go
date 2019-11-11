package model

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/derekparker/trie"
	"gopkg.in/yaml.v3"
)

// NodeMeta comment
type NodeMeta struct {
	Config nodeConfig
	Size   int64
	IsDir  bool
	//isEnum bool
	//isConfig bool
}

// Create trie model
func Create(dirpath string) *trie.Trie {
	model := trie.New()

	err := filepath.Walk(dirpath,
		func(curpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			var config nodeConfig

			_, file := filepath.Split(curpath)
			if file == "dim1.yaml" {
				config, err = loadConfig(curpath)
				if err != nil {
					return err
				}
			}
			meta := NodeMeta{
				Config: config,
				Size:   info.Size(),
				IsDir:  info.IsDir()}
			fmt.Println(curpath, meta)
			model.Add(curpath, meta)
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return model
}

// Args interface
type Args interface{}

type nodeConfig struct {
	Name    string          `yaml:"name"`
	Enum    []string        `yaml:"enum,flow"`
	ArgsMap map[string]Args `yaml:"args,inline"`
}

func loadConfig(filepath string) (nodeConfig, error) {
	var config nodeConfig

	filedata, err := ioutil.ReadFile(filepath)
	if err != nil {
		return config, err
	}
	yaml.Unmarshal(filedata, &config)
	fmt.Printf("Yaml values:\n%v\n", config)
	return config, nil
}
