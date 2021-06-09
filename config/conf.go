package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
	"runtime"
)

type Conf struct {
	Etcd struct{
		EndPoints []string `json:"endpoints"`
	} `json:"etcd"`
}

var Config *Conf

func init() {
	_, filename, _, _ := runtime.Caller(0) // get current filepath in runtime
	filepath := path.Dir(filename)
	by, _ := ioutil.ReadFile(filepath + "/" + "config.yaml")
	_ = yaml.Unmarshal(by, &Config)
}
func main() {
	fmt.Println(Config.Etcd)
}