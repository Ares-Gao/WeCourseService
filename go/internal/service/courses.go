package service

import (
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"io/ioutil"
	"net/http"
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

func GetSemester(UserName, PassWord string) string {
	return GetSemesterWithLogin(UserName, PassWord, "", "")
}

func GetSemesterWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	conf := ReadConfig()
	result := SemesterResult{Type: "semester"}
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		fmt.Println("ERROR_SEMESTER_LOGIN: ", err.Error())
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	time.Sleep(time.Duration(1000 * time.Millisecond))
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/courseTableForStd.action", nil)
	if err != nil {
		fmt.Println("ERROR_SEMESTER_7: ", err.Error())
	}
	resp3, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_SEMESTER_8: ", err.Error())
	}
	defer resp3.Body.Close()

	content, err := ioutil.ReadAll(resp3.Body)
	if err != nil {
		fmt.Println("ERROR_SEMESTER_9: ", err.Error())
	}
	ids, semesterID, ok := getCourseTableParams(string(content))
	if ok {
		result.Data = append(result.Data, SemesterStruct{SemesterID: semesterID, Ids: ids, Current: true})
	}

	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/logout.action", nil)
	if err == nil {
		resp4, err := client.Do(req)
		if err == nil {
			defer resp4.Body.Close()
		}
	}

	js, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return ""
	}
	return B2S(js)
}

func getCourseTableParams(html string) (string, string, bool) {
	idsReg := regexp.MustCompile(`bg\.form\.addInput\(form,\s*"ids",\s*"([^"]+)"`)
	idsMatch := idsReg.FindStringSubmatch(html)
	if len(idsMatch) < 2 {
		return "", "", false
	}

	semesterPatterns := []string{
		`name=["']semester\.id["'][^>]*value=["']([^"']+)["']`,
		`semesterCalendar\(\{[^}]*value:\s*"([^"]+)"`,
		`semesterCalendar\(\{[^}]*value:\s*'([^']+)'`,
		`bg\.form\.addInput\(form,\s*"semester\.id",\s*"([^"]+)"`,
	}
	for _, pattern := range semesterPatterns {
		semesterReg := regexp.MustCompile(pattern)
		semesterMatch := semesterReg.FindStringSubmatch(html)
		if len(semesterMatch) >= 2 && semesterMatch[1] != "" {
			return idsMatch[1], semesterMatch[1], true
		}
	}
	return "", "", false
}

func cleanJSText(text string) string {
	text = strings.ReplaceAll(text, `"+periodInfo+"`, "")
	text = strings.ReplaceAll(text, `\"`, `"`)
	return text
}

func GetCourse(UserName, PassWord string) string {
	return GetCourseWithLogin(UserName, PassWord, "", "")
}

func GetCourseWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	cacheKey := loginType + ":" + authServerURL + ":" + UserName
	value, found := c.Get(cacheKey)
	if found {
		//fmt.Print("Using Cache")
		if value.(string) != "" {
			return value.(string)
		}
	}
	//readcache in there
	// 获取用户名和密码
	conf := ReadConfig()
	myCourses = nil
	teachers = nil

	myAllCourseResult.Type = "allcourse"
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		fmt.Println("ERROR_8: LOGIN Failed")
		myAllCourseResult.Data = myCourses
		js, _ := json.MarshalIndent(myAllCourseResult, "", "\t")
		return B2S(js)
	}
	time.Sleep(1000 * time.Millisecond)
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/courseTableForStd.action", nil)
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
	content, err := ioutil.ReadAll(resp3.Body)
	if err != nil {
		fmt.Println("ERROR_11: ", err.Error())
		//return
	}

	temp := string(content)
	ids, semesterID, ok := getCourseTableParams(temp)
	if !ok {
		fmt.Println("ERROR_12: GET ids Failed")
		//return
	}

	formValues := make(url.Values)
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
		course.CourseName = cleanJSText(courseStr[2])
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
	c.Set(cacheKey, cachestr, cache.DefaultExpiration)
	value_check, found_check := c.Get(cacheKey)
	if found_check {
		//fmt.Print("Using Cache")
		if value_check.(string) == "" {
			c.Set(cacheKey, cachestr, cache.DefaultExpiration)
		}
	}
	return cachestr

}
