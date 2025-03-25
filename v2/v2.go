package v2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"text/template"

	//    "reflect"
	"os/exec"
	"strconv"
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
	Exec      string   `yaml:"exec"`
	Diag      string   `yaml:"diag"`
	Dir       string   `yaml:"dir"`
	ForVars   []string `yaml:"for-vars"`
	Condition string   `yaml:"condition"`
}

type Block struct {
	Name        string                   `yaml:"name"`
	Desc        string                   `yaml:"desc"`
	CommandsRaw []map[string]interface{} `yaml:"commands"`
	Commands    []Command                `yaml:"Commands"`
	Vars        map[string]interface{}   `yaml:"vars"`
	Dir         string                   `yaml:"dir"`
	Children    []Block                  `yaml:"children"`
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
	processBlock(block)
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
		case map[string]interface{}:

			varCfg := VarCfg{}

			if typeVal["from_env"] != nil {
				varCfg.FromEnv = typeVal["from_env"].(string)
				if envVal := os.Getenv(varCfg.FromEnv); len(envVal) > 0 {
					varCfg.Value = envVal
				}
			}

			if typeVal["from_command"] != nil {
				varCfg.FromCommand = typeVal["from_command"].(string)

				var output bytes.Buffer

				execConfig := ExecConfig{
					Stdout: &output,
				}

				execConfig.Cmd = "/bin/bash"
				execConfig.Args = []string{"-c", varCfg.FromCommand}

				if exit := execCommand(execConfig); exit == 0 {
					lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")

					if len(lines) > 1 {
						varCfg.ListValue = lines
					} else {
						varCfg.Value = lines[0]
					}
				}
			}

			if typeVal["default"] != nil {
				varCfg.Default = typeVal["default"].(string)
			}

			if len(varCfg.Value) == 0 && len(varCfg.ListValue) == 0 && len(varCfg.Default) > 0 {
				varCfg.Value = varCfg.Default
			}

			VarCfgs[varName] = varCfg

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
				Value: strconv.FormatUint(typeVal, 10),
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

func render(tmpl string, varCfgs map[string]VarCfg) string {

	/*
	   Converting from the template format established in the perl version
	*/
	t1, err := tt.Parse(fixupRe.ReplaceAllString(tmpl, "{{ .$1.Value }}"))
	if err != nil {
		panic(err)
	}

	var renderBuf bytes.Buffer

	t1.Execute(&renderBuf, varCfgs)

	return renderBuf.String()
}

func initBlockCommands(block *Block) {
	for _, command := range block.CommandsRaw {

		/* All this because for-vars can be a string referncing a list or list */
		Command := Command{}

		if command["exec"] != nil {
			Command.Exec = command["exec"].(string)
		}
		if command["diag"] != nil {
			Command.Diag = command["diag"].(string)
		}
		if command["dir"] != nil {
			Command.Dir = command["dir"].(string)
		}
		if command["for-vars"] != nil {
			switch typeVal := command["for-vars"].(type) {
			case []interface{}:

				for _, elem := range typeVal {

					Command.ForVars = append(Command.ForVars, elem.(string))

				}

			case string:

				if list := VarCfgs[typeVal]; list.ListValue != nil {
					Command.ForVars = list.ListValue
				}
			default:
				fmt.Printf("I don't know about type %T in for-vars!\n", typeVal)
			}
		} else {
			Command.ForVars = []string{"1"}
		}

		block.Commands = append(block.Commands, Command)

	}

	block.CommandsRaw = nil
}

func processBlock(block Block) {

	initVars(block.Vars)

	config := ExecConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if len(block.Dir) > 0 {
		config.Dir = block.Dir
	} else {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot get current working directory \n")
			return
		} else {
			config.Dir = dir
		}

	}

	initBlockCommands(&block)
	runCommandsWithConfig(block.Commands, config)
}

type ExecConfig struct {
	Cmd    string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
}

func runCommandsWithConfig(commands []Command, config ExecConfig) {
	for _, command := range commands {

		/* local copy. I don't know why this is needed
		   to stop changes made to config outside this
		   scope.  Isn't Go suppose to pass structs
		   by value? */
		execConfig := config

		if len(command.Dir) > 0 {
			execConfig.Dir = render(command.Dir, VarCfgs)
		}

		/* This behaves slightly different from the perl version
		       1. Diag wont override Exec and both can run if defined
			   2. Diag and Exec will both be looped with for-vars

		*/
		for index, value := range command.ForVars {

			varCfgs := map[string]VarCfg{}

			maps.Copy(varCfgs, VarCfgs)
			maps.Copy(varCfgs, map[string]VarCfg{"index": {Value: strconv.Itoa(index)}, "var": {Value: value}})

			if len(command.Diag) > 0 {
				execConfig.Cmd = "/usr/bin/echo"
				execConfig.Args = []string{render(command.Diag, varCfgs)}

				execCommand(execConfig)
			}

			if len(command.Exec) > 0 {
				execConfig.Cmd = "/bin/bash"
				execConfig.Args = []string{"-c", render(command.Exec, varCfgs)}

				execCommand(execConfig)
			}
		}

	}
}

func execCommand(config ExecConfig) int {

	cmd := exec.Command(config.Cmd, config.Args...)
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr
	cmd.Dir = config.Dir

	err := cmd.Run()
	if err != nil {

		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		} else {
			return 1
		}

	}

	return 0
}
