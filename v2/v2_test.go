package v2

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, e error, s string) {
	if e != nil {
		t.Errorf("%s - %v", s, e)
	}
}

func createTestConfig(t *testing.T, config string) (*os.File, []byte, error) {

	data := []byte(config)

	tcfg, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating temp cfg file")

	_, err = tcfg.Write(data)
	check(t, err, "Error writing to temp cfg file")

	yamlFile, err := os.Open(tcfg.Name())
	check(t, err, "Error opening temp yaml file")

	yamlData, err := io.ReadAll(yamlFile)
	check(t, err, "Error reading yaml data")

	return tcfg, yamlData, nil
}

type DexTest struct {
	Name       string
	Config     string
	Dexfile    DexFile2
	MenuOut    string
	Blockpath  []string
	Commands   []Command
	CommandOut string
}

func TestParseConfigFile(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Hello",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description`,
			Dexfile: DexFile2{
				Version: 2,
				Blocks: []Block{
					{
						Name: "hello",
						Desc: "this is a command description",
					},
				},
			},
		},
		{
			Name: "Hello Children",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description
    children:
      - name: start
        desc: start the server
      - name: stop
        desc: stop the server
      - name: restart
        desc: restart the server
`,
			Dexfile: DexFile2{
				Version: 2,
				Blocks: []Block{
					{
						Name: "hello",
						Desc: "this is a command description",
						Children: []Block{
							{
								Name: "start",
								Desc: "start the server",
							},
							{
								Name: "stop",
								Desc: "stop the server",
							},
							{
								Name: "restart",
								Desc: "restart the server",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)
		check(t, err, "config file not found")

		assert.Equal(t, test.Dexfile, dex_file)

	}

}

func TestDisplayMenu(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Hello",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description`,
			MenuOut: "hello                   : this is a command description\n",
		},
		{
			Name: "Hello Children",
			Config: `---
version: 2
blocks:
  - name: hello
    desc: this is a command description
    children:
      - name: start
        desc: start the server
      - name: stop
        desc: stop the server
      - name: restart
        desc: restart the server
`,
			MenuOut: `hello                   : this is a command description
    start                   : start the server
    stop                    : stop the server
    restart                 : restart the server
`,
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, _ := ParseConfig(yamlData)

		var output bytes.Buffer
		displayMenu(&output, dex_file.Blocks, 0)

		assert.Equal(t, test.MenuOut, output.String())

	}

}

func TestResolveBlock(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Nested Command",
			Config: `---
version: 2
blocks:
  - name: server 
    desc: this is a command description
    children:
      - name: start
        desc: start the server
      - name: stop
        desc: stop the server
      - name: restart
        desc: restart the server
        commands: 
          - exec: systemctl restart server
          - exec: touch /.restarted 
`,
			Blockpath: []string{"server", "restart"},
			Commands:  []Command{{Exec: "systemctl restart server"}, {Exec: "touch /.restarted"}},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)

		check(t, err, "Error parsing config")

		block_cmds, err := resolveCmdToCodeblock(dex_file.Blocks, test.Blockpath)

		check(t, err, "Error resolving command")

		assert.Equal(t, test.Commands, block_cmds)

	}
}

func TestCommands(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Nested Command",
			Config: `---
version: 2
blocks:
  - name: hello_world 
    desc: this is a command description
    commands: 
       - exec: echo "hello world"
`,
			Blockpath:  []string{"hello_world"},
			CommandOut: "hello world\n",
			//Commands:  []Command{{Exec: "systemctl restart server"}, {Exec: "touch /.restarted"}},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)

		check(t, err, "Error parsing config")

		block_cmds, err := resolveCmdToCodeblock(dex_file.Blocks, test.Blockpath)

		check(t, err, "Error resolving command")

		var output bytes.Buffer

		config := CommandConfig{
			Stdout: &output,
			Stderr: &output,
		}

		runCommandsWithConfig(block_cmds, config)

		assert.Equal(t, test.CommandOut, output.String())

	}
}

func TestVars(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Nested Command",
			Config: `---
version: 2
vars: 
  string_var: "hi there"
  int_var: 2 
  list_var:
    - these
    - those
blocks:
  - name: hello_world 
    desc: this is a command description
    commands: 
       - exec: echo "hello world"
`,
			Blockpath:  []string{"hello_world"},
			CommandOut: "hello world\n",
			//Commands:  []Command{{Exec: "systemctl restart server"}, {Exec: "touch /.restarted"}},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)

		check(t, err, "Error parsing config")

		initVars(dex_file.Vars)

		t.Logf("%s", VarCfgs)
		t.Logf("string var is %s", VarCfgs["string_var"].Value)

		//commands, err := resolveCmdToCodeblock(dex_file.Blocks, test.Blockpath)

		//check(t, err, "Error resolving command")

	}
}
