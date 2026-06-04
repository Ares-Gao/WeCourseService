package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Config struct {
	SchoolName                string
	MangerType                string
	MangerURL                 string
	CalendarFirst             string
	SocketPort                int
	LoginType                 string
	AuthServerURL             string
	ServiceURL                string
	AuthServerAutoCaptcha     bool
	AuthServerCaptchaRetries  int
	DdddOcrOnnxRuntimeLibPath string
	DdddOcrModelPath          string
	DdddOcrDictPath           string
	DdddOcrDetModelPath       string
	DdddOcrUseCustomModel     bool
	ClassTimeSlots            []ClassTimeSlot
	CalendarTimezone          string
	CalendarName              string
}

type ClassTimeSlot struct {
	Start string
	End   string
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
