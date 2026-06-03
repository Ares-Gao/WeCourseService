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

type TeacherStruct struct {
	CourseID      string
	CourseName    string
	CourseCredit  string
	CourseTeacher string
}

func GetTeacher(UserName, PassWord string) string {
	// 获取用户名和密码
	conf := ReadConfig()
	USERNAME := UserName
	PASSWORD := PassWord
	var teachers []TeacherStruct
	var myTeacher TeacherStruct
	var teacherResult TeacherResult
	// Cookie自动维护
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		//return
	}
	var client http.Client
	client.Jar = cookieJar
	teacherResult.Type = "teacher"
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
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/courseTableForStd.action", nil)
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
	ids, semesterID, ok := getCourseTableParams(temp)
	if !ok {
		fmt.Println("ERROR_12: GET ids Failed")
		//return
	}

	formValues = make(url.Values)
	formValues.Set("ignoreHead", "1")
	formValues.Set("showPrintAndExport", "1")
	formValues.Set("setting.kind", "std")
	formValues.Set("startWeek", "")
	formValues.Set("semester.id", semesterID)
	formValues.Set("ids", ids)
	req, err = http.NewRequest(http.MethodPost, conf.MangerURL+"eams/courseTableForStd!courseTable.action", strings.NewReader(formValues.Encode()))
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
	if !strings.Contains(temp, "课表格式说明") {
		fmt.Println("ERROR_16: Get Courses Failed")
		//return
	}
	reg3 := regexp.MustCompile(`(?i)<td>(\d)</td>\s*<td>([:alpha:].+)</td>\s*<td>(.+)</td>\s*<td>((\d)|(\d\.\d))</td>\s*<td>\s*<a href=.*\s.*\s.*\s.*>.*</a>\s*</td>\s*<td>(.*)</td>`)
	reg4 := regexp.MustCompile(`(?i)<td>([^>]*)</td>`)
	reg5 := regexp.MustCompile(`(?i)>([^>]*)</a>`)
	teanchersStr := reg3.FindAllStringSubmatch(temp, -1)
	for _, teacherStr := range teanchersStr {
		teacher := reg4.FindAllStringSubmatch(teacherStr[0], -1)
		courseid := reg5.FindAllStringSubmatch(teacherStr[0], -1)
		myTeacher.CourseID = courseid[0][1]
		myTeacher.CourseName = teacher[2][1]
		myTeacher.CourseCredit = teacher[3][1]
		myTeacher.CourseTeacher = teacher[4][1]
		teachers = append(teachers, myTeacher)
	}
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/logout.action", nil)
	if err != nil {
		fmt.Println("ERROR_17: ", err.Error())
		//return
	}

	resp5, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_18: ", err.Error())
		//return
	}
	defer resp5.Body.Close()
	teacherResult.Data = teachers
	js, err := json.MarshalIndent(teacherResult, "", "\t")
	return B2S(js)

}

