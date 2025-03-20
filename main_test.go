package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, e error, s string) {
	if e != nil {
		t.Errorf("%s - %v", s, e)
	}
}

func TestFindConfigFile(t *testing.T) {
	_, err := find_config_file()

	if err == nil {
		t.Error("No error on config file not found")
	}

	f, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating cfg file")

	defer os.Remove(f.Name())

	f2, err := os.CreateTemp("", "dex-test")
	check(t, err, "Error creating second cfg file")

	defer os.Remove(f2.Name())

	config_file_locations = []string{"not-exists.yml", f.Name(), f2.Name()}

	cfg, err := find_config_file()
	check(t, err, "config file not found")

	assert.Equal(t, cfg, f.Name())

	os.Remove(f.Name())

	cfg2, err := find_config_file()
	check(t, err, "config file not found")

	assert.Equal(t, cfg2, f2.Name())

	t.Logf("cfg file %s", cfg)

}

type MenuData struct {
	Name  string
	Desc  string
	Depth int
}
type ParseTest struct {
	Name    string
	Config  string
	Dexfile DexFile
	// Menu   []MenuData
}

func TestParseConfigFile(t *testing.T) {

	tests := []ParseTest{
		{
			Name: "Hello",
			Config: `---
                     - name: hello
                       desc: this is a command description`,
			Dexfile: DexFile{
				{
					Name: "hello",
					Desc: "this is a command description",
				},
			},
			//Menu: []MenuData{
			//	{
			//		Name:  "hello",
			//		Desc:  "hello",
			//		Depth: 1,
			//	},
			//},
		},
		{
			Name: "Hello Children",
			Config: `---
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
			Dexfile: DexFile{
				{
					Name: "hello",
					Desc: "this is a command description",
					Children: DexFile{
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
	}

	for _, test := range tests {
		t.Logf("test %s\n", test.Name)

		data := []byte(test.Config)

		tcfg, err := os.CreateTemp("", "dex-test")
		check(t, err, "Error creating temp cfg file")

		defer os.Remove(tcfg.Name())

		_, err = tcfg.Write(data)
		check(t, err, "Error writing to temp cfg file")

		dex_file, err := parse_config_file(tcfg.Name())
		check(t, err, "config file not found")

		assert.Equal(t, dex_file, test.Dexfile)

		t.Logf("%v", dex_file)

	}

}
