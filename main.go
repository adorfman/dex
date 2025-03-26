package main

import (
	"fmt"
	"io"
	"os"

	v1 "dex/v1"
	v2 "dex/v2"
)

// Paths to search for dex files.
var configFileLocations = []string{"dex.yaml", "dex.yml", ".dex.yaml", ".dex.yml"}

/*
1. Try to locate a dex file, throw an error and exit if there is no config file.
2. Load the content of the dex file
3. Attempt to parse the dex file as v1 YAML.
*/
func main() {

	/* Find the name of the dex file we're using. */
	filename, err := findConfigFile()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dexData, err := loadDexFile(filename)

	if err != nil {
		os.Exit(1)
	}

	if dexFile, err := v1.ParseConfig(dexData); err == nil {
		v1.Run(dexFile, os.Args)
	} else if dexFile, err := v2.ParseConfig(dexData); err == nil {
		v2.Run(dexFile, os.Args)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadDexFile(filename string) ([]byte, error) {

	fileContent, err := os.Open(filename)
	if err != nil {
		fmt.Printf("yamlFile.Get err #%v ", err)
		return []byte{}, err
	}

	dexData, err := io.ReadAll(fileContent)

	if err != nil {
		fmt.Printf("yamlFile.Get err #%v ", err)
		return []byte{}, err
	}

	return dexData, err
}

/*
Search through the config_files array and return the first

	dex file that exists.
*/
func findConfigFile() (string, error) {

	/* DEX_FILE env takes priority */
	if dexFileEnv := os.Getenv("DEX_FILE"); len(dexFileEnv) > 0 {
		if _, err := os.Stat(dexFileEnv); err == nil {
			return dexFileEnv, nil
		}
	}

	for _, filename := range configFileLocations {
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	return "", fmt.Errorf("no dex file was found.  Searched %v", configFileLocations)
}
