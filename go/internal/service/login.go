package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

func GetUserLogin(UserName, PassWord string) string {
	// 获取用户名和密码
	USERNAME := UserName
	PASSWORD := PassWord
	var myLogin LoginResult
	myLogin.Type = "login"
	conf := ReadConfig()
	// Cookie自动维护
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		//return ("ERROR_0: ", err.Error())
		//return
	}
	var client http.Client
	client.Jar = cookieJar

	// 第一次请求
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/login.action", nil)
	if err != nil {
		//return ("ERROR_1: ", err.Error())
		//return
	}

	resp1, err := client.Do(req)
	if err != nil {
		//return ("ERROR_2: ", err.Error())
		//return
	}
	defer resp1.Body.Close()

	content, err := ioutil.ReadAll(resp1.Body)
	if err != nil {
		//return ("ERROR_3: ", err.Error())
		//return
	}

	temp := string(content)
	if !strings.Contains(temp, "CryptoJS.SHA1(") {
		//return ("ERROR_4: GET Failed")
		//return
	}

	// 对密码进行SHA1哈希
	temp = temp[strings.Index(temp, "CryptoJS.SHA1(")+15 : strings.Index(temp, "CryptoJS.SHA1(")+52]
	PASSWORD = temp + PASSWORD
	bytes := sha1.Sum([]byte(PASSWORD))
	PASSWORD = hex.EncodeToString(bytes[:])
	formValues := make(url.Values)
	formValues.Set("username", USERNAME)
	formValues.Set("password", PASSWORD)
	formValues.Set("session_locale", "zh_CN")
	time.Sleep(time.Duration(1000 * time.Millisecond))
	req, err = http.NewRequest(http.MethodPost, conf.MangerURL+"eams/login.action", strings.NewReader(formValues.Encode()))
	if err != nil {
		//return ("ERROR_5: ", err.Error())
		//return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")
	resp2, err := client.Do(req)
	if err != nil {
		//return ("ERROR_6: ", err.Error())
		//return
	}
	defer resp2.Body.Close()

	content, err = ioutil.ReadAll(resp2.Body)
	if err != nil {
		//return ("ERROR_7: ", err.Error())
		//return
	}

	temp = string(content)
	if !strings.Contains(temp, "<a href=\"/eams/security/my.action\" target=\"_blank\" title=\"查看详情\" style=\"color:#ffffff\">") {
		myLogin.Data = "登录失败"
		js, _ := json.MarshalIndent(myLogin, "", "\t")
		return B2S(js) //Write cache in here
		//return ("ERROR_8: LOGIN Failed")
		//return
	}
	myLogin.Data = "登录成功"
	js, err := json.MarshalIndent(myLogin, "", "\t")
	return B2S(js) //Write cache in here
}

