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
	check(t, err, "Error creating temp file")

	defer os.Remove(f.Name())

	config_file_locations = []string{"not-exists.yml", f.Name()}

	cfg, err := find_config_file()
	check(t, err, "config file not found")

	assert.Equal(t, cfg, f.Name())
	t.Logf("cfg file %s", cfg)

}
