package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
)

// Paths to search for dex files.
var config_file_locations = []string{"dex.yaml", "dex.yml", ".dex.yaml", ".dex.yml"}

func config_files() []string {
	return config_file_locations
}

/* Struct to turn the YAML file into */
type DexFile []struct {
	Name     string   `yaml:"name"`
	Desc     string   `yaml:"desc"`
	Commands []string `yaml:"shell"`
	Children DexFile  `yaml:"children"`
}

/*
Main function:
 1. Load the config file, throw an error and exit if there is no config file.
 2. Turn the YAML structure from the config file into a DexFile struct.
 3. If there was no commands to run, display the menu of commands the DexFile knows about.
 4. If there was a command to run, find it and run it.  If it's invalid, say so and display the menu.
*/
func main() {

	/* Find the name of the dex file we're using. */
	filename, err := find_config_file()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dex_file, err := parse_config_file(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	/* No commands asked for: show menu and exit */
	if len(os.Args) == 1 {
		display_menu(os.Stdout, dex_file, 0)
		os.Exit(0)
	}

	/* No commands were found from the arguments the user passed: show error, menu and exit */
	commands, err := resolve_cmd_to_codeblock(dex_file, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: No commands were found at %v\n\nSee the menu:\n", os.Args[1:])
		display_menu(os.Stderr, dex_file, 0)
		os.Exit(1)
	}

	/* Found commands: run them */
	run_commands(commands)

}

func parse_config_file(filename string) (DexFile, error) {

	/* Load the contents of the dex file */
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read data from config file (%v): %w", filename, err)
	}

	/* Get the YAML structure from the contents of the dex file */
	var dex_file DexFile
	if err := yaml.Unmarshal([]byte(data), &dex_file); err != nil {
		return nil, fmt.Errorf("could not parse YAML from file (%v): %s", filename, err)
	}

	return dex_file, nil
}

/*
Search through the config_files array and return the first

	dex file that exists.
*/
func find_config_file() (string, error) {
	config_files := config_files()

	for _, filename := range config_files {
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	return "", errors.New(fmt.Sprintf("No dex file was found.  Searched %v", config_files))
}

/*
Display the menu by recursively processing each element of the DexFile and

	showing the name and description for the command.  Children are indented with
	4 spaces.
*/
func display_menu(w io.Writer, dex_file DexFile, indent int) {
	for _, elem := range dex_file {

		fmt.Fprintf(w, "%s%-24v: %v\n", strings.Repeat(" ", indent*4), elem.Name, elem.Desc)

		if len(elem.Children) >= 1 {
			display_menu(w, elem.Children, indent+1)
		}
	}
}

/*
Find the list of commands to run for a given command path.

	For example, cmd = [ 'foo', 'bar', 'blee' ] would check if 'foo' is a valid command,
	then call itself with the child DexFile of foo, and cmd = ['bar', 'blee'].  Then bar's
	child DexFile would be called with [ 'blee' ] and return the list of commands.
*/
func resolve_cmd_to_codeblock(dex_file DexFile, cmds []string) ([]string, error) {
	for _, elem := range dex_file {
		if elem.Name == cmds[0] {
			if len(cmds) >= 2 {
				return resolve_cmd_to_codeblock(elem.Children, cmds[1:])
			} else {
				return elem.Commands, nil
			}
		}
	}
	return []string{}, errors.New("Could not find command.")
}

/*
Given a list of commands, run them.

	Uses bash so that quoting, shell expansion, etc works.
	Writes the stdout/stderr as one would expect.
*/
func run_commands(commands []string) {
	for _, command := range commands {
		cmd := exec.Command("/bin/bash", "-c", command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to run command: ", err)
		}
	}
}
