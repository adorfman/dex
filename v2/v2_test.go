package v2

import (
	//"bytes"
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
	Name      string
	Config    string
	Dexfile   DexFile2
	MenuOut   string
	Blockpath []string
	Commands  []string
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
