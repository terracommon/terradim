package model

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
	//"github.com/zclconf/go-cty/cty/gocty"
	"gopkg.in/yaml.v3"
)

// TerradimConfig is the marshalled config for a terrdim file (ex. n1.yaml)
type TerradimConfig struct {
	Path   string
	Config nodeConfig
}

// TerradimConfigMap hold paths for terradim enum dirs
type TerradimConfigMap map[string]*TerradimConfig

// NodeMeta comment
type NodeMeta struct {
	Dirname    string
	Basename   string
	Size       int64
	IsFileRoot bool
	IsDir      bool
	IsEnum     bool
	IsConfig   bool
}

// BuildConfig contains all build congig details
type BuildConfig struct {
	ConfigMap      TerradimConfigMap
	FileRootPrefix string
	FileOutPrefix  string
}

// ModelConfig list enumerable dirs
var ModelConfig = TerradimConfigMap{
	"dim1": &TerradimConfig{Config: nodeConfig{}},
	"dim2": &TerradimConfig{Config: nodeConfig{}},
}

// Create tree model
func Create(dirpath string) (*Tree, BuildConfig) {
	var (
		parent     *Node
		parentMeta NodeMeta
	)
	outDir := "live"
	dirname, basename := filepath.Split(dirpath)
	buildConfig := BuildConfig{
		ConfigMap:      ModelConfig,
		FileRootPrefix: dirpath,
		FileOutPrefix:  dirpath[:len(dirname)] + outDir,
	}
	model := NewTree()
	lastDirname := dirpath
	err := filepath.Walk(dirpath,
		func(curpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			dirname, basename = filepath.Split(curpath)
			meta := NodeMeta{
				Dirname:  dirname,
				Basename: basename,
				Size:     info.Size(),
			}
			if curpath != dirpath && dirname != lastDirname {
				lastDirname = dirname
				parent, _ = model.Find(dirname[:len(dirname)-1])
				if parent.meta == nil {
					parentMeta = NodeMeta{}
					meta.IsFileRoot = true
					fmt.Printf("IS FILE ROOT: %+v\n", meta)
				} else {
					parentMeta = parent.Meta().(NodeMeta)
					fmt.Printf("Parent Meta: %+v\n", parentMeta)
				}
			}
			if info.IsDir() {
				meta.IsDir = true
				if _, ok := ModelConfig[basename]; ok == true {
					meta.IsEnum = true
					buildConfig.ConfigMap[basename].Path = curpath
				} else {
					dirParts := strings.SplitN(basename, "_", 2)
					_, ok := ModelConfig[dirParts[0]]
					if ok && dirParts[1] == "config" {
						meta.IsConfig = true
					}
				}
			} else {
				fileParts := strings.SplitN(basename, ".", 2)
				_, ok := ModelConfig[fileParts[0]]
				if ok && fileParts[1] == "yaml" {
					meta.IsEnum = true
					meta.IsConfig = true
					buildConfig.ConfigMap[fileParts[0]].Config = loadNodeConfig(curpath)
				}
			}
			if parentMeta.IsConfig == true {
				meta.IsConfig = true
			}
			fmt.Println(curpath, meta)
			_, _ = model.Insert(curpath, meta)
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return model, buildConfig
}

type nodeConfig struct {
	Name    string   `yaml:"name"`
	Outfile string   `yaml:"outfile"`
	Enum    []string `yaml:"enum,flow"`
}

func loadNodeConfig(curpath string) nodeConfig {
	var config nodeConfig
	filedata, err := ioutil.ReadFile(curpath)
	if err != nil {
		panic(err)
	}
	yaml.Unmarshal(filedata, &config)
	return config
}

func loadConfig(curPath string) (nodeConfig, error) {
	var config nodeConfig
	_, fileName := filepath.Split(curPath)

	scope := &lang.Scope{
		BaseDir: ".",
	}
	exprTxt := fmt.Sprintf(`yamldecode(file("%v"))`, curPath)
	expr, parseDiags := hclsyntax.ParseExpression([]byte(exprTxt), fileName, hcl.Pos{Line: 1, Column: 1})
	if parseDiags.HasErrors() {
		for _, diag := range parseDiags {
			fmt.Println(diag.Error())
		}
		panic("ParseExpression Error")
	}
	got, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
	if diags.HasErrors() {
		for _, diag := range diags {
			fmt.Printf("%s: %s", diag.Description().Summary, diag.Description().Detail)
		}
		panic("EvalExpr Error")
	}

	value := got
	fmt.Printf("lang type: %v - %v - %#v - %v\n", exprTxt, expr, value, value.Type().FriendlyName())

	//for friendly name got.AsValueMap()["args"].AsValueMap()["foo"].Type().FriendlyName()

	//argsInt := struct{ Enum []string }{
	//	Enum: make([]string, 0),
	//}
	//err := gocty.FromCtyValue(value, &argsInt)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("ArgsInt: ", argsInt)

	filedata, err := ioutil.ReadFile(curPath)
	if err != nil {
		return config, err
	}
	yaml.Unmarshal(filedata, &config)
	fmt.Printf("Yaml values:\n%v\n", config)
	return config, nil
}

// Write model to file
func Write(t *Tree, config BuildConfig) error {
	root := t.Root()
	fmt.Printf("Root: %+v\n", root)
	WalkSubtree(root, buildFunc)
	return nil
}

// buildFunc is a Tree WalkFunc for writing model to filesystem
func buildFunc(node *Node, err error) (bool, error) {
	if node.Meta() == nil {
		return true, nil
	}
	meta := node.Meta().(NodeMeta)
	fmt.Println("Walk: ", meta.Dirname, meta.Basename)

	if meta.IsEnum && meta.IsDir {
		// Iterate through enum subtrees and write config files
	} else if meta.IsConfig == false {
		// Write dir or file
	}

	return true, nil
}
