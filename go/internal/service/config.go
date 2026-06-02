package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Config struct {
	SchoolName    string
	MangerType    string
	MangerURL     string
	CalendarFirst string
	SocketPort    int
}

func ReadConfig() Config {
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Println(err)
	}
	var conf Config
	err = json.Unmarshal(data, &conf)
	if err != nil {
		fmt.Println(err)
	}
	return conf
}

