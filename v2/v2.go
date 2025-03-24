package v2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"text/template"

	//    "reflect"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
)

var VarCfgs = map[string]VarCfg{}

type VarCfg struct {
	Value       string
	ListValue   []string
	FromCommand string `yaml:"from_command"`
	FromEnv     string `yaml:"from_env"`
	Default     string `yaml:"default"`
}

type ListVarCfg struct {
	Value       []string
	FromCommand string   `yaml:"from_command"`
	FromEnv     string   `yaml:"from_env"`
	Default     []string `yaml:"default"`
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
	Blocks  []Block                `yaml:"blocks"`
}

/*
1. If there was no commands to run, display the menu of commands the DexFile knows about.
2. If there was a command to run, find it and run it.  If it's invalid, say so and display the menu.
*/
func Run(dexFile DexFile2, args []string) {

	/* No commands asked for: show menu and exit */

	if len(args) == 1 {
		displayMenu(os.Stdout, dexFile.Blocks, 0)
		os.Exit(0)
	}

	block, err := resolveCmdToCodeblock(dexFile.Blocks, args[1:])
	//
	// /* No commands were found from the arguments the user passed: show error, menu and exit */
	//
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: No commands were found at %v\n\nSee the menu:\n", args[1:])
		displayMenu(os.Stderr, dexFile.Blocks, 0)
		os.Exit(1)
	}

	initVars(dexFile.Vars)

	//
	// /* Found commands: run them */
	processBloack(block)
}

/*
Attempt to parse the YAML content into DexFile format
*/
func ParseConfig(configData []byte) (DexFile2, error) {

	var dexFile DexFile2

	if err := yaml.Unmarshal([]byte(configData), &dexFile); err != nil {
		return DexFile2{}, err
	} else if dexFile.Version != 2 {
		return DexFile2{}, errors.New("incorrect version number")
	}

	return dexFile, nil
}

/*
Display the menu by recursively processing each element of the DexFile and

	showing the name and description for the command.  Children are indented with
	4 spaces.
*/
func displayMenu(w io.Writer, blocks []Block, indent int) {
	for _, elem := range blocks {

		fmt.Fprintf(w, "%s%-24v: %v\n", strings.Repeat(" ", indent*4), elem.Name, elem.Desc)

		if len(elem.Children) >= 1 {
			displayMenu(w, elem.Children, indent+1)
		}
	}
}

func resolveCmdToCodeblock(blocks []Block, cmds []string) (Block, error) {
	for _, elem := range blocks {
		if elem.Name == cmds[0] {
			if len(cmds) >= 2 {
				return resolveCmdToCodeblock(elem.Children, cmds[1:])
			} else {
				return elem, nil
			}
		}
	}
	return Block{}, errors.New("could not find command")
}

func initVars(varMap map[string]interface{}) {
	for varName, value := range varMap {

		switch typeVal := value.(type) {
		case []interface{}:

			VarCfgs[varName] = VarCfg{
				ListValue: []string{},
			}

			for _, elem := range typeVal {

				entry := VarCfgs[varName]
				entry.ListValue = append(entry.ListValue, elem.(string))

				VarCfgs[varName] = entry
			}

		case uint64:

			VarCfgs[varName] = VarCfg{
				//Value: string(typeVal),
				Value: fmt.Sprintf("%d", typeVal),
			}

		case string:

			VarCfgs[varName] = VarCfg{
				Value: typeVal,
			}
		default:
			fmt.Printf("I don't know about type %T for %s!\n", typeVal, varName)
		}
	}
}

var fixupRe = regexp.MustCompile(`\[%\s*(\S+)\s*%\]`)
var tt = template.New("variable_parser")

func render(tmpl string) string {

	t1, err := tt.Parse(fixupRe.ReplaceAllString(tmpl, "{{ .$1.Value }}"))
	if err != nil {
		panic(err)
	}

	var renderBuf bytes.Buffer

	t1.Execute(&renderBuf, VarCfgs)

	return renderBuf.String()
}

func processBloack(block Block) {

	initVars(block.Vars)
	runCommands(block.Commands)
}

type CommandConfig struct {
	Stdout io.Writer
	Stderr io.Writer
}

func runCommands(commands []Command) {

	config := CommandConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	runCommandsWithConfig(commands, config)
}

func runCommandsWithConfig(commands []Command, config CommandConfig) {
	for _, command := range commands {
		cmd := exec.Command("/bin/bash", "-c", render(command.Exec))
		cmd.Stdout = config.Stdout
		cmd.Stderr = config.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to run command: ", err)
		}
	}
}
