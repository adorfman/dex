package main

import (
 "fmt"
 "os" 
 //"io/ioutil"
 "io"
 "github.com/goccy/go-yaml" 
 "github.com/mitchellh/mapstructure"
)

type DexFile []struct {
    Name       string  `yaml:"name"`
    Desc       string  `yaml:"desc"`
    Commands []string  `yaml:"shell"`
    Children   DexFile `yaml:"children"`
}

func main() {
 obj := make(map[string]interface{})

 yamlFile, err := os.Open("dex-test.yaml")
 yamlData, err := io.ReadAll(yamlFile)
 //yamlFile, err := ioutil.ReadFile("dex-test.yaml")
 if err != nil {
  fmt.Printf("yamlFile.Get err #%v ", err)
 }

 if err := yaml.Unmarshal([]byte(yamlData), &obj); err != nil {

    var dex_file DexFile 

    _, err = yamlFile.Seek(0, io.SeekStart)
    if err != nil {
       fmt.Printf("Seek err #%v ", err)   
    }

    if err := yaml.Unmarshal([]byte(yamlData), &dex_file); err != nil {
        fmt.Fprintln(os.Stderr, fmt.Errorf("Could not parse YAML from file: %w", err))
        os.Exit(1)
    } else {
      fmt.Println(dex_file[0].Commands) 
    }

 } else {
    fmt.Println(obj)
    fmt.Println(obj["version"])

    type V2 struct {
       Version int     `mapstructure:"version"`
       Key     string   `mapstructure:"key"` 
       Foo     map[string]string   `mapstructure:"foo"` 
    }

    var result V2
    // result := &v2{}
    err := mapstructure.Decode(obj, &result)

    // name, ok := carDetails["name"]
    // if !ok {
    //   return errors.New("missing name")
    // }


    if err != nil {
       panic(err)
    }
    fmt.Println(result.Version) 
    //fmt.Printf("%#v", result)
 }

}
