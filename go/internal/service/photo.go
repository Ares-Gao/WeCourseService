package service

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

func GetPhoto(UserName, PassWord string) string {
	// 获取用户名和密码
	USER := UserName
	PASS := PassWord
	conf := ReadConfig()
	var myPhotoResult PhotoResult
	// Cookie自动维护
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		//return
	}
	myPhotoResult.Type = "photo"
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
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/showSelfAvatar.action?user.name="+USER, nil)
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

	temp = base64.StdEncoding.EncodeToString(content)
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
	myPhotoResult.Data = "data:image/jpg;base64," + temp
	js, err := json.MarshalIndent(myPhotoResult, "", "\t")
	return B2S(js) //Write cache in here
}

