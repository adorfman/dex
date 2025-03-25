package v2

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, e error, s string) error {
	if e != nil {
		t.Errorf("%s - %v", s, e)
		return e
	}
	return nil
}

func createTestConfig(t *testing.T, config string) (*os.File, []byte, error) {

	data := []byte(config)

	tDexFile, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating temp cfg file")

	_, err = tDexFile.Write(data)
	check(t, err, "Error writing to temp cfg file")

	yamlFile, err := os.Open(tDexFile.Name())
	check(t, err, "Error opening temp yaml file")

	yamlData, err := io.ReadAll(yamlFile)
	check(t, err, "Error reading yaml data")

	return tDexFile, yamlData, nil
}

func setupTestBlock(t *testing.T, test DexTest) (Block, *os.File, error) {

	tDexFile, yamlData, _ := createTestConfig(t, test.Config)

	dexFile, err := ParseConfig(yamlData)

	if err := check(t, err, "Error parsing config"); err != nil {
		return Block{}, nil, err
	}

	/* reset VarCfgs */
	VarCfgs = map[string]VarCfg{}

	initVars(dexFile.Vars)

	block, err := resolveCmdToCodeblock(dexFile.Blocks, test.Blockpath)

	if err := check(t, err, "Error resolving command"); err != nil {
		return Block{}, nil, err
	}

	initVars(block.Vars)

	initBlockCommands(&block)

	return block, tDexFile, nil
}

type DexTest struct {
	Name         string
	Config       string
	Dexfile      DexFile2
	MenuOut      string
	Blockpath    []string
	Commands     []Command
	CommandsRaw  []map[string]interface{}
	CommandOut   string
	ExpectedVars map[string]VarCfg
	Custom       func(t *testing.T, test DexTest, opts map[string]interface{})
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
						Name:        "hello",
						Desc:        "this is a command description",
						Commands:    nil,
						CommandsRaw: nil,
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
        commands:
          - exec: systemctl start server
      - name: stop
        desc: stop the server
        commands:
          - exec: systemctl stop server
            dir: /home/slice
      - name: restart
        desc: restart the server
        commands:
          - exec: systemctl stop server
          - exec: systemctl start server
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
								//Commands: []Command{
								//	{
								//		Exec: "systemctl start server",
								//	},
								//},
								Commands:    nil,
								CommandsRaw: []map[string]interface{}{{"exec": "systemctl start server"}},
							},
							{
								Name: "stop",
								Desc: "stop the server",
								//Commands: []Command{
								//	{
								//		Exec: "systemctl stop server",
								//	},
								//},
								Commands:    nil,
								CommandsRaw: []map[string]interface{}{{"exec": "systemctl stop server", "dir": "/home/slice"}},
							},
							{
								Name: "restart",
								Desc: "restart the server",
								//Commands: []Command{
								//	{
								//		Exec: "systemctl stop server",
								//	},
								//	{
								//		Exec: "systemctl start server",
								//	},
								//},
								Commands: nil,
								CommandsRaw: []map[string]interface{}{
									{"exec": "systemctl stop server"},
									{"exec": "systemctl start server"}},
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
			CommandsRaw: []map[string]interface{}{
				{"exec": "systemctl restart server"},
				{"exec": "touch /.restarted"}},
		},
	}

	for _, test := range tests {

		tcfg, yamlData, _ := createTestConfig(t, test.Config)

		defer os.Remove(tcfg.Name())

		dex_file, err := ParseConfig(yamlData)

		check(t, err, "Error parsing config")

		block, err := resolveCmdToCodeblock(dex_file.Blocks, test.Blockpath)

		check(t, err, "Error resolving command")

		assert.Equal(t, test.CommandsRaw, block.CommandsRaw)

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

		block, err := resolveCmdToCodeblock(dex_file.Blocks, test.Blockpath)

		check(t, err, "Error resolving command")

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
		}

		initBlockCommands(&block)
		runCommandsWithConfig(block.Commands, config)

		assert.Equal(t, test.CommandOut, output.String())

	}
}

func TestVars(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Global Vars",
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
			ExpectedVars: map[string]VarCfg{
				"string_var": {
					Value: "hi there",
				},
				"int_var": {
					Value: "2",
				},
				"list_var": {
					ListValue: []string{
						"these",
						"those",
					},
				},
			},
		},
		{
			Name: "Block Vars",
			Config: `---
version: 2
vars: 
  global_string: "foobar"

blocks:
  - name: block_vars 
    desc: this is a command description
    vars: 
      string_var: "from block"
      int_var: 3 
      list_var:
        - one 
        - two 
    commands: 
       - exec: echo "hello world"
  - name: other_block 
    desc: this is a command description
    vars: 
      string_var: "other local block var"

`,
			Blockpath:  []string{"block_vars"},
			CommandOut: "hello world\n",
			ExpectedVars: map[string]VarCfg{
				"global_string": {
					Value: "foobar",
				},
				"string_var": {
					Value: "from block",
				},
				"int_var": {
					Value: "3",
				},
				"list_var": {
					ListValue: []string{
						"one",
						"two",
					},
				},
			},
		},
		{
			Name: "Vars From Env",
			Config: `---
version: 2
vars:
  global_string:
    from_env: TESTENV 
  not_set:
    from_env: TESTENV_UNSET 
    default: fizzbizz 

blocks:
  - name: block_vars
    desc: this is a command description
`,
			Blockpath: []string{"block_vars"},
			ExpectedVars: map[string]VarCfg{
				"global_string": {
					FromEnv: "TESTENV",
					Value:   "from env!",
				},
				"not_set": {
					FromEnv: "TESTENV_UNSET",
					Default: "fizzbizz",
					Value:   "fizzbizz",
				},
			},
		},
		{
			Name: "Vars From command",
			Config: `---
version: 2
vars:
  command_string:
    from_command: echo "c var" 
  command_list:
    from_command: echo -en "foo\nbar\nbazz" 

blocks:
  - name: block_vars
    desc: this is a command description
`,
			Blockpath: []string{"block_vars"},
			ExpectedVars: map[string]VarCfg{
				"command_string": {
					FromCommand: "echo \"c var\"",
					Value:       "c var",
				},
				"command_list": {
					FromCommand: "echo -en \"foo\\nbar\\nbazz\"",
					ListValue:   []string{"foo", "bar", "bazz"},
				},
			},
		},
	}

	for _, test := range tests {

		os.Setenv("TESTENV", "from env!")

		_, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		//t.Logf("%v", VarCfgs)
		//t.Logf("%v", test.ExpectedVars)
		assert.True(t, reflect.DeepEqual(test.ExpectedVars, VarCfgs))
	}
}

func TestRenderedCommand(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Global Vars",
			Config: `---
version: 2
vars: 
  string_var: "hi there"
blocks:
  - name: hello_world 
    desc: this is a command description
    commands: 
       - exec: echo "[%string_var%]"
`,
			Blockpath:  []string{"hello_world"},
			CommandOut: "hi there\n",
		},
		{
			Name: "Block Vars",
			Config: `---
version: 2
vars:
  global_string: "foobar"

blocks:
  - name: block_vars
    desc: this is a command description
    vars:
      string_var: "from block"
      int_var: 3
    commands:
       - exec: echo "[% global_string %] [% string_var %] [% int_var %]"
`,
			Blockpath:  []string{"block_vars"},
			CommandOut: "foobar from block 3\n",
		},
		{
			Name: "Diag",
			Config: `---
version: 2
vars:
  global_string: "foobar"

blocks:
  - name: diag_command 
    desc: this is a command description
    vars:
      string_var: "from block"
      int_var: 4
    commands:
       - diag: "[% global_string %] [% string_var %] [% int_var %]"
`,
			Blockpath:  []string{"diag_command"},
			CommandOut: "foobar from block 4\n",
		},
	}

	for _, test := range tests {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
		}

		runCommandsWithConfig(block.Commands, config)

		assert.Equal(t, test.CommandOut, output.String())

		//t.Logf("%s", VarCfgs)
		//t.Logf("string var is %s", VarCfgs["string_var"].Value)

		//assert.True(t, reflect.DeepEqual(test.ExpectedVars, VarCfgs))
	}
}

func TestCommandDir(t *testing.T) {

	tests := []DexTest{
		{
			Name: "Block dir",
			Config: `---
version: 2
blocks:
  - name: change_dir 
    dir:  ".." 
    desc: this is a command description
    commands: 
       - exec: echo $(pwd)
`,
			Blockpath: []string{"change_dir"},
			Custom: func(t *testing.T, test DexTest, opts map[string]interface{}) {

				output := opts["ouput"].(bytes.Buffer)

				path, _ := os.Getwd()

				parentDir := filepath.Dir(path) + "\n"

				assert.Equal(t, parentDir, output.String())
			},
		},
		{
			Name: "Command Dir",
			Config: `---
version: 2
blocks:
  - name: change_dir 
    desc: this is a command description
    commands: 
       - exec: echo $(pwd)
         dir:  ".." 
       - exec: echo $(pwd)

`,
			Blockpath: []string{"change_dir"},
			Custom: func(t *testing.T, test DexTest, opts map[string]interface{}) {

				output := opts["ouput"].(bytes.Buffer)

				path, _ := os.Getwd()

				parentDir := filepath.Dir(path) + "\n" + path + "\n"

				assert.Equal(t, parentDir, output.String())
			},
		},
		//		{
		//			Name: "Block Vars",
		//			Config: `---
		//version: 2
		//vars:
		//  global_string: "foobar"
		//
		//blocks:
		//  - name: block_vars
		//    desc: this is a command description
		//    vars:
		//      string_var: "from block"
		//      int_var: 3
		//    commands:
		//       - exec: echo "[% global_string %] [% string_var %] [% int_var %]"
		//`,
		//			Blockpath:  []string{"block_vars"},
		//			CommandOut: "foobar from block 3\n",
		//		},
		//		{
		//			Name: "Diag",
		//			Config: `---
		//version: 2
		//vars:
		//  global_string: "foobar"
		//
		//blocks:
		//  - name: diag_command
		//    desc: this is a command description
		//    vars:
		//      string_var: "from block"
		//      int_var: 4
		//    commands:
		//       - diag: "[% global_string %] [% string_var %] [% int_var %]"
		//`,
		//			Blockpath:  []string{"diag_command"},
		//			CommandOut: "foobar from block 4\n",
		//		},
	}

	for _, test := range tests[1:2] {

		block, tDexFile, err := setupTestBlock(t, test)

		defer os.Remove(tDexFile.Name())

		if err := check(t, err, "error setting up test"); err != nil {
			continue
		}

		var output bytes.Buffer

		config := ExecConfig{
			Stdout: &output,
			Stderr: &output,
			Dir:    block.Dir,
		}

		t.Logf("%v", block.Commands)
		runCommandsWithConfig(block.Commands, config)

		//t.Logf("%s", output.String())
		test.Custom(t, test, map[string]interface{}{"ouput": output})

	}
}
