package model

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
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
	Dirname  string
	Basename string
	Size     int64
	IsDir    bool
	IsEnum   bool
	IsConfig bool
}

// BuildConfig contains all build congig details
type BuildConfig struct {
	ConfigMap      TerradimConfigMap
	FileRootPrefix string
	FileOutPrefix  string
	PathSeparator  string
}

type buildData map[string]interface{}

// ModelConfig list enumerable dirs
var ModelConfig = TerradimConfigMap{
	"dim1": &TerradimConfig{Config: nodeConfig{}},
	"dim2": &TerradimConfig{Config: nodeConfig{}},
}

// Create tree model
func Create(dirpath string) (*Tree, *BuildConfig) {
	var (
		parent     *Node
		node       *Node
		parentMeta NodeMeta
	)
	outDir := "live"
	dirname, basename := filepath.Split(dirpath)
	model := NewTree()
	buildConfig := &BuildConfig{
		ConfigMap:      ModelConfig,
		FileRootPrefix: dirpath,
		FileOutPrefix:  dirpath[:len(dirname)] + outDir,
		PathSeparator:  model.Separator(),
	}
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
				} else {
					parentMeta = parent.Meta().(NodeMeta)
				}
			}
			if info.IsDir() {
				meta.IsDir = true
				if _, ok := ModelConfig[basename]; ok == true {
					if err != nil {
						panic(err)
					}
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
			//fmt.Println(curpath, meta)
			node, _ = model.Insert(curpath, meta)
			fmt.Println(curpath, meta, node.prefix, node.key)
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
func Write(t *Tree, config *BuildConfig) error {
	root := t.Root()
	fmt.Printf("Root: %+v\n", root)
	WalkSubtree(root, buildFunc, buildData{"buildConfig": config})
	return nil
}

// buildFunc is a Tree WalkFunc for writing model to filesystem
func buildFunc(node *Node, data interface{}) (bool, error) {
	if node.Meta() == nil {
		return true, nil
	}
	var (
		err       error
		writePath string
	)
	meta := node.Meta().(NodeMeta)
	key := node.Key()
	path := node.Path()
	dataMap := data.(buildData)
	buildConfig, ok := dataMap["buildConfig"].(*BuildConfig)
	if ok == false {
		return false, errors.New("buildFunc: Must pass in BuildConfig as data")
	}
	//fmt.Println("Walk: ", meta.Dirname, meta.Basename, dataMap)
	if meta.IsConfig {
		if meta.IsDir == false {
			//fmt.Printf("Config %s\n", writePath)
		}
		return false, nil
	}
	if meta.IsEnum && meta.IsDir {
		keyNum, err := strconv.Atoi(key[len(key)-1:])
		if err != nil {
			return false, err
		}

		for _, enum := range buildConfig.ConfigMap[key].Config.Enum {
			dataMap[key] = enum
			for dim := range buildConfig.ConfigMap {
				if _, ok := dataMap[dim].(string); ok && dim != key {
					dimLevel, err := strconv.Atoi(dim[len(dim)-1:])
					if err != nil {
						panic(err)
					}
					if dimLevel > keyNum {
						delete(dataMap, dim)
					}
				}
			}
			writePath, err = copyToDst(path, &dataMap)
			if err != nil {
				return true, err
			}
			fmt.Printf("Write: %s   ->   %s\n", path, writePath)

			for _, child := range node.Children() {
				WalkSubtree(child, buildFunc, dataMap)
			}
		}
		return false, nil
	}
	writePath, err = copyToDst(path, &dataMap)
	if err != nil {
		return true, err
	}
	fmt.Printf("Write: %s   ->   %s\n", path, writePath)
	return true, nil
}

func createWritePath(src string, data *buildData) (dst string, err error) {
	dataMap := *data
	buildConfig, ok := dataMap["buildConfig"].(*BuildConfig)
	if ok == false {
		return "", errors.New("buildFunc: Must pass in BuildConfig as data")
	}

	dst = buildConfig.FileOutPrefix + src[len(buildConfig.FileRootPrefix):]
	sep := buildConfig.PathSeparator

	for dim := range buildConfig.ConfigMap {
		if val, ok := dataMap[dim]; ok {
			dst = strings.Replace(dst, fmt.Sprintf("%s%s%s", sep, dim, sep),
				fmt.Sprintf("%s%s%s", sep, val, sep), 1)

			if dir, file := filepath.Split(dst); file == dim {
				dst = fmt.Sprintf("%s%s", dir, val)
			}
		}
	}
	return
}

func copyToDst(path string, data *buildData) (dst string, err error) {
	dst, err = createWritePath(path, data)
	if err != nil {
		return
	}

	err = Copy(path, dst)
	return
}

func collectDimConfigs(enumNode *Node, data *buildData) ([]string, error) {
	dims := []string{}
	dataMap := *data
	ext := ".yaml"
	key := enumNode.Key()
	configNode := enumNode.getChild(key + "_config")
	dimBase := key[:len(key)-1]
	dimLevel, err := strconv.Atoi(key[len(key)-1:])
	if err != nil {
		return nil, err
	}

	if child := configNode.getChild(dataMap[key].(string) + ext); child != nil {
		dims = append(dims, child.Path())
	}
	if child := configNode.getChild(dataMap[key].(string)); child != nil && dimLevel == 2 {
		if gChild := configNode.getChild(dataMap[key].(string) + ext); gChild != nil {
			dims = append(dims, gChild.Path())
		}
		gridKey := fmt.Sprintf("%s_%s%s", key, dataMap[dimBase+"1"], ext)
		if gChild := configNode.getChild(dataMap[key].(string) + ext); gChild != nil {
			dims = append(dims, gChild.Path())
		}
		if gChild := configNode.getChild(gridKey); gChild != nil {
			dims = append(dims, gChild.Path())
		}
	}
	return dims, nil
}

func mergeDimConfigs(configPaths []string) (string, error) {
	for i, val := range configPaths {
		configPaths[i] = fmt.Sprintf(`yamldecode(file("%s"))`, val)
	}

	exprTxt := fmt.Sprintf(`yamlencode(merge(%s))`, strings.Join(configPaths, ","))
	// TODO: Figure out how to set filename
	fileName := "config"
	expr, parseDiags := hclsyntax.ParseExpression([]byte(exprTxt), fileName, hcl.Pos{Line: 1, Column: 1})
	if parseDiags.HasErrors() {
		for _, diag := range parseDiags {
			fmt.Println(diag.Error())
		}
		return "", errors.New("mergeDimConfigs: ParseExpression Error")
	}

	scope := &lang.Scope{
		BaseDir: ".",
	}
	got, diags := scope.EvalExpr(expr, cty.String)
	if diags.HasErrors() {
		for _, diag := range diags {
			fmt.Printf("%s: %s", diag.Description().Summary, diag.Description().Detail)
		}
		panic("EvalExpr Error")
	}

	var config string
	err := gocty.FromCtyValue(got, config)
	return config, err
}

func writeDimConfig(enumNode *Node, data *buildData) (string, error) {
	dataMap := *data
	buildConfig, ok := dataMap["buildConfig"].(*BuildConfig)
	if ok == false {
		return "", errors.New("buildFunc: Must pass in BuildConfig as data")
	}

	configPaths, err := collectDimConfigs(enumNode, data)
	if err != nil {
		return "", err
	}

	dimConfig, err := mergeDimConfigs(configPaths)
	if err != nil {
		return "", err
	}

	configName := buildConfig.ConfigMap[enumNode.Key()].Config.Outfile
	path := fmt.Sprintf("%s%s%s", enumNode.Path(), enumNode.Sep(), configName)
	dst, err := createWritePath(path, data)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(dst, []byte(dimConfig), info.Mode())
	return dst, err
}
