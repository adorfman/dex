package v2

import (
	//	"errors"
	//	"fmt"
	//	"io"
	//	"os"
	//	"os/exec"
	//	"strings"

	"github.com/goccy/go-yaml"
)

type Var struct {
	FromCommand string `yaml:"from_command"`
	FromEnv     string `yaml:"from_env"`
	Default     string `yaml:"default"`
}

type Command struct {
	Exec      string                 `yaml:"exec"`
	Diag      string                 `yaml:"diag"`
	Dir       string                 `yaml:"dir"`
	ForVars   map[string]interface{} `yaml:"for-vars"`
	Condition string                 `yaml:"condition"`
}

type Block struct {
	Name     string                 `yaml:"name"`
	Desc     string                 `yaml:"desc"`
	Commands []Command              `yaml:"commands"`
	Vars     map[string]interface{} `yaml:"vars"`
	Dir      string                 `yaml:"dir"`
	Children []Block                `yaml:"children"`
}
type DexFile2 struct {
	Version int                    `yaml:"version"`
	Vars    map[string]interface{} `yaml:"vars"`
	Blocks  []Block
}

/*
1. If there was no commands to run, display the menu of commands the DexFile knows about.
2. If there was a command to run, find it and run it.  If it's invalid, say so and display the menu.
*/
func Run(dexFile DexFile2, args []string) {

	// /* No commands asked for: show menu and exit */
	//
	//	if len(args) == 1 {
	//		displayMenu(os.Stdout, dexFile, 0)
	//		os.Exit(0)
	//	}
	//
	// /* No commands were found from the arguments the user passed: show error, menu and exit */
	// commands, err := resolveCmdToCodeblock(dexFile, args[1:])
	//
	//	if err != nil {
	//		fmt.Fprintf(os.Stderr, "Error: No commands were found at %v\n\nSee the menu:\n", args[1:])
	//		displayMenu(os.Stderr, dexFile, 0)
	//		os.Exit(1)
	//	}
	//
	// /* Found commands: run them */
	// runCommands(commands)
}

/*
Attempt to parse the YAML content into DexFile format
*/
func ParseConfig(configData []byte) (DexFile2, error) {

	var dexFile DexFile2

	if err := yaml.Unmarshal([]byte(configData), &dexFile); err != nil {
		return DexFile2{}, err
	}

	return dexFile, nil
}
