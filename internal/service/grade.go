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

type GradeStruct struct {
	CourseID     string
	CourseName   string
	CourseTerm   string
	CourseCredit string
	CourseGrade  string
	GradePoint   string
}

//TODO:Fill Struct Array
func GetGrade(UserName, PassWord string) string {
	// 获取用户名和密码
	conf := ReadConfig()
	USERNAME := UserName
	PASSWORD := PassWord
	var grades []GradeStruct
	var myGrade GradeStruct
	var gradeResult GradeResult
	// Cookie自动维护
	cookieJar, err := cookiejar.New(nil)
	gradeResult.Type = "grade"
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
	req, err = http.NewRequest(http.MethodPost, conf.MangerURL+"eams/teach/grade/course/person!historyCourseGrade.action?projectType=MAJOR", strings.NewReader(formValues.Encode()))
	if err != nil {
		fmt.Println("ERROR_13: ", err.Error())
		//return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")
	resp4, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_14: ", err.Error())
		//return
	}
	defer resp4.Body.Close()

	content, err = ioutil.ReadAll(resp4.Body)
	if err != nil {
		fmt.Println("ERROR_15: ", err.Error())
		//return
	}

	temp = string(content)
	reg3 := regexp.MustCompile(`(?i)<tr>[\s\S]*?</tr>`)
	reg4 := regexp.MustCompile(`(?i)<td.*>([^>]*)</td>`)
	reg5 := regexp.MustCompile(`(?i)<sup.*>([^>]*)</sup>`)
	gradeStr := reg3.FindAllStringSubmatch(temp, -1)
	gradeStr = append(gradeStr[:0], gradeStr[0+1:]...)
	gradeStr = append(gradeStr[:0], gradeStr[0+1:]...)
	for _, tempStr := range gradeStr {
		fuck := reg4.FindAllStringSubmatch(tempStr[0], -1)
		myGrade.CourseTerm = strings.Trim(fuck[0][1], "\n")
		myGrade.CourseID = strings.Trim(fuck[1][1], "\n")
		myGrade.CourseCredit = strings.Trim(fuck[4][1], "\n")
		bodyclass := reg5.FindAllStringSubmatch(tempStr[0], -1)
		if len(bodyclass) != 0 {
			myGrade.CourseName = bodyclass[0][1]
		} else {
			myGrade.CourseName = strings.Trim(fuck[3][1], "\t\r\n")
		}
		myGrade.CourseGrade = strings.Trim(fuck[len(fuck)-2][1], "\t\n")
		myGrade.GradePoint = strings.Trim(fuck[len(fuck)-1][1], "\t\n")
		grades = append(grades, myGrade)
	}
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/logout.action", nil)
	if err != nil {
		fmt.Println("ERROR_17: ", err.Error())
	}

	resp5, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_18: ", err.Error())
		//return
	}
	defer resp5.Body.Close()
	gradeResult.Data = grades
	js, err := json.MarshalIndent(gradeResult, "", "\t")
	return B2S(js)
}

