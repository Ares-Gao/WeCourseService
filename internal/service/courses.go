package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 课程持续时间，周几第几节
type CourseTime struct {
	DayOfTheWeek int
	TimeOfTheDay int
}

// 课程信息
type Course struct {
	CourseID    string
	CourseName  string
	RoomID      string
	RoomName    string
	Weeks       string
	CourseTimes []CourseTime
}

var USERNAME, PASSWORD string
var myCourses []Course
var teachers []TeacherStruct
var myTeacher TeacherStruct
var myAllCourseResult CourseResult
var c = cache.New(1*time.Hour, 10*time.Minute)

func B2S(bs []byte) string {
	ba := []byte{}
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(ba)
}
func GetTeacherObj() []TeacherStruct {
	return teachers
}
func GetCourse(UserName, PassWord string) string {
	value, found := c.Get(UserName)
	if found {
		//fmt.Print("Using Cache")
		if value.(string) != "" {
			return value.(string)
		}
	}
	//readcache in there
	// 获取用户名和密码
	conf := ReadConfig()
	USERNAME := UserName
	PASSWORD := PassWord
	myCourses = nil
	teachers = nil

	myAllCourseResult.Type = "allcourse"
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
	if !strings.Contains(temp, "bg.form.addInput(form,\"ids\",\"") {
		fmt.Println("ERROR_12: GET ids Failed")
		//return
	}

	temp = temp[strings.Index(temp, "bg.form.addInput(form,\"ids\",\"")+29 : strings.Index(temp, "bg.form.addInput(form,\"ids\",\"")+50]
	ids := temp[:strings.Index(temp, "\");")]
	formValues = make(url.Values)
	formValues.Set("ignoreHead", "1")
	formValues.Set("showPrintAndExport", "1")
	formValues.Set("setting.kind", "std")
	formValues.Set("startWeek", "")
	formValues.Set("semester.id", "30")
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
	reg1 := regexp.MustCompile(`TaskActivity\(actTeacherId.join\(','\),actTeacherName.join\(','\),"(.*)","(.*)\(.*\)","(.*)","(.*)","(.*)",null,null,assistantName,"",""\);((?:\s*index =\d+\*unitCount\+\d+;\s*.*\s)+)`)
	reg2 := regexp.MustCompile(`\s*index =(\d+)\*unitCount\+(\d+);\s*`)
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
	coursesStr := reg1.FindAllStringSubmatch(temp, -1)
	for _, courseStr := range coursesStr {
		var course Course
		course.CourseID = courseStr[1]
		course.CourseName = courseStr[2]
		course.RoomID = courseStr[3]
		course.RoomName = courseStr[4]
		course.Weeks = courseStr[5]
		for _, indexStr := range strings.Split(courseStr[6], "table0.activities[index][table0.activities[index].length]=activity;") {
			if !strings.Contains(indexStr, "unitCount") {
				continue
			}
			var courseTime CourseTime
			courseTime.DayOfTheWeek, _ = strconv.Atoi(reg2.FindStringSubmatch(indexStr)[1])
			courseTime.TimeOfTheDay, _ = strconv.Atoi(reg2.FindStringSubmatch(indexStr)[2])
			course.CourseTimes = append(course.CourseTimes, courseTime)
		}
		myCourses = append(myCourses, course)
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
	myAllCourseResult.Data = myCourses
	js, err := json.MarshalIndent(myAllCourseResult, "", "\t")
	cachestr := B2S(js)
	c.Set(UserName, cachestr, cache.DefaultExpiration)
	value_check, found_check := c.Get(UserName)
	if found_check {
		//fmt.Print("Using Cache")
		if value_check.(string) == "" {
			c.Set(UserName, cachestr, cache.DefaultExpiration)
		}
	}
	return cachestr

}

