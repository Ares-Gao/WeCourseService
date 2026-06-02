package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"
)

func GetWeekTime(startTime string) string {
	var timeReuslt TimeResult
	timeReuslt.Type = "week"
	timeTemplate := "2006-01-02 15:04:05"
	now, _ := time.Parse(timeTemplate, startTime+" 00:00:00")
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	nowtime := time.Now()
	week := (float64(nowtime.Unix()) - float64(monday.Unix())) / 604800
	week = math.Ceil(week)
	timeReuslt.Data = strconv.Itoa(int(week))
	js, _ := json.MarshalIndent(timeReuslt, "", "\t")
	return B2S(js)
}

func GetWeekTimeOld(serverIP, startTime string) string {
	//此方法依赖服务器计算教学周，已弃用
	url := "http://" + serverIP + "/api/nowweek.php?date=" + startTime
	var client http.Client
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err)
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	return string(content)
}

