package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var USER, PASS string

type StudentStruct struct {
	FullName    string
	EnglishName string
	Sex         string
	StartTime   string
	EndTime     string
	SchoolYear  string
	Type        string
	System      string
	Specialty   string
	Class       string
}

func GetAccount(UserName, PassWord string) string {
	// 获取用户名和密码
	var myStudent StudentStruct
	var myAccountResult AccountResult
	USER := UserName
	PASS := PassWord
	conf := ReadConfig()
	myAccountResult.Type = "account"
	// Cookie自动维护
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		//return
	}
	var client http.Client
	client.Jar = cookieJar

	// 第一次请求
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/login.action", nil)
	if err != nil {
		fmt.Println("ERROR_1: ", err.Error())
		//return
	}

	resp1, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_2: ", err.Error())
		//return
	}
	defer resp1.Body.Close()

	content, err := ioutil.ReadAll(resp1.Body)
	if err != nil {
		fmt.Println("ERROR_3: ", err.Error())
		//return
	}

	temp := string(content)
	if !strings.Contains(temp, "CryptoJS.SHA1(") {
		fmt.Println("ERROR_4: GET Failed")
		//return
	}

	// 对密码进行SHA1哈希
	temp = temp[strings.Index(temp, "CryptoJS.SHA1(")+15 : strings.Index(temp, "CryptoJS.SHA1(")+52]
	PASS = temp + PASS
	bytes := sha1.Sum([]byte(PASS))
	PASS = hex.EncodeToString(bytes[:])

	formValues := make(url.Values)
	formValues.Set("username", USER)
	formValues.Set("password", PASS)
	formValues.Set("session_locale", "zh_CN")
	time.Sleep(1000 * time.Millisecond)
	req, err = http.NewRequest(http.MethodPost, conf.MangerURL+"eams/login.action", strings.NewReader(formValues.Encode()))
	if err != nil {
		fmt.Println("ERROR_5: ", err.Error())
		//return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")
	resp2, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_6: ", err.Error())
		//return
	}
	defer resp2.Body.Close()

	content, err = ioutil.ReadAll(resp2.Body)
	if err != nil {
		fmt.Println("ERROR_7: ", err.Error())
		//return
	}

	temp = string(content)
	if !strings.Contains(temp, "<a href=\"/eams/security/my.action\" target=\"_blank\" title=\"查看详情\" style=\"color:#ffffff\">") {
		fmt.Println(temp)
		fmt.Println("ERROR_8: LOGIN Failed")
		//return
	}
	time.Sleep(1000 * time.Millisecond)
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/stdDetail.action", nil)
	if err != nil {
		fmt.Println("ERROR_9: ", err.Error())
		//return
	}

	resp3, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_10: ", err.Error())
		//return
	}

	defer resp3.Body.Close()
	content, err = ioutil.ReadAll(resp3.Body)
	if err != nil {
		fmt.Println("ERROR_11: ", err.Error())
		//return
	}

	temp = string(content)
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/logout.action", nil)
	if err != nil {
		fmt.Println("ERROR_12: ", err.Error())
		//return
	}

	resp5, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_13: ", err.Error())
		//return
	}
	defer resp5.Body.Close()
	reg := regexp.MustCompile(`(?i)<td>([^>]*)</td>`)
	stuinfo := reg.FindAllStringSubmatch(temp, -1)
	//fmt.Println(stuinfo)
	myStudent.FullName = stuinfo[0][1]
	myStudent.EnglishName = stuinfo[1][1]
	myStudent.Sex = stuinfo[2][1]
	myStudent.SchoolYear = stuinfo[4][1]
	myStudent.Type = stuinfo[5][1] + "(" + stuinfo[14][1] + ")"
	myStudent.StartTime = stuinfo[11][1]
	myStudent.EndTime = stuinfo[12][1]
	myStudent.System = stuinfo[8][1]
	myStudent.Specialty = stuinfo[9][1]
	myStudent.Class = stuinfo[18][1]
	myAccountResult.Data = myStudent
	js, err := json.MarshalIndent(myAccountResult, "", "\t")
	return B2S(js)
}

