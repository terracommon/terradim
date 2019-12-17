package model

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/lang"
	"github.com/spf13/viper"
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

type nodeConfig struct {
	Name    string   `yaml:"name"`
	Outfile string   `yaml:"outfile"`
	Enum    []string `yaml:"enum,flow"`
}

// ModelConfig list enumerable dirs
var ModelConfig = TerradimConfigMap{
	"dim1": &TerradimConfig{Config: nodeConfig{}},
	"dim2": &TerradimConfig{Config: nodeConfig{}},
}

// Create tree model
func Create(srcpath, dstpath string) (*Tree, *BuildConfig) {
	var (
		parent     *Node
		parentMeta NodeMeta
	)
	dirname, basename := filepath.Split(srcpath)
	model := NewTree()
	buildConfig := &BuildConfig{
		ConfigMap:      ModelConfig,
		FileRootPrefix: srcpath,
		FileOutPrefix:  dstpath,
		PathSeparator:  model.Separator(),
	}
	lastDirname := srcpath
	err := filepath.Walk(srcpath,
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
			if curpath != srcpath && dirname != lastDirname {
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
						return err
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

			_, _ = model.Insert(curpath, meta)
			if viper.GetBool("verbose") == true {
				fmt.Printf("Insert: %s - %+v\n", curpath, meta)
			}
			return nil
		})
	if err != nil {
		panic(err)
	}
	return model, buildConfig
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

// Write model to file
func Write(t *Tree, config *BuildConfig) error {
	root := t.Root()
	WalkSubtree(root, buildFunc, buildData{"buildConfig": config})
	return nil
}

// buildFunc is a Tree WalkFunc for writing model to filesystem
func buildFunc(node *Node, data interface{}) (bool, error) {
	if node.Meta() == nil {
		return true, nil
	}
	var err error
	meta := node.Meta().(NodeMeta)
	key := node.Key()
	path := node.Path()
	dataMap := data.(buildData)
	buildConfig, ok := dataMap["buildConfig"].(*BuildConfig)
	if ok == false {
		return false, errors.New("buildFunc: Must pass in BuildConfig as data")
	}

	if meta.IsConfig {
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
			_, err = copyToDst(path, &dataMap)
			if err != nil {
				return true, err
			}

			_, err = writeDimConfig(node, &dataMap)
			if err != nil {
				return true, err
			}

			for _, child := range node.Children() {
				WalkSubtree(child, buildFunc, dataMap)
			}
		}
		return false, nil
	}
	_, err = copyToDst(path, &dataMap)
	if err != nil {
		return true, err
	}
	return true, nil
}

func createWritePath(src string, data *buildData) (dst string, err error) {
	fmt.Println("SRC", src)
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
	//TODO: if flag --verbose fmt.Printf("Write: %s  ->  %s\n", path, dst)
	if viper.GetBool("verbose") == true {
		fmt.Printf("Write: %s  ->  %s\n", path, dst)
	}
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
		if gChild := child.getChild(dataMap[key].(string) + ext); gChild != nil {
			dims = append(dims, gChild.Path())
		}
		gridKey := fmt.Sprintf("%s_%s%s", dataMap[key], dataMap[dimBase+"1"], ext)
		if gChild := child.getChild(dataMap[key].(string) + ext); gChild != nil {
			dims = append(dims, gChild.Path())
		}
		if gChild := child.getChild(gridKey); gChild != nil {
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
		return "", errors.New("mergeDimConfigs: EvalExpr Error")
	}

	var config string
	err := gocty.FromCtyValue(got, &config)
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

	enumPath := enumNode.Path()
	configName := buildConfig.ConfigMap[enumNode.Key()].Config.Outfile
	configPath := fmt.Sprintf("%s%s%s", enumPath, enumNode.Sep(), configName)
	dst, err := createWritePath(configPath, data)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(enumPath)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(dst, []byte(dimConfig), info.Mode())

	// TODO: if flag --verbose fmt.Printf("Write Config: ->  %s\n", writePath)
	if viper.GetBool("verbose") == true {
		fmt.Printf("Write Config: ->  %s\n", dst)
	}
	return dst, err
}
