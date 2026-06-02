package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type WeekCourseNew struct {
	CourseName  string
	TeacherName string
	RoomName    string
	CourseTimes []CourseTime
}

var myWeekCourseNew []WeekCourseNew
var tmpCourseNew WeekCourseNew
var myCourseResultNew WeekCourseResultNew

func GetWeekCourseNew(UserName string, PassWord string, WeekDay int) string {
	myWeekCourseNew = nil
	myCourseResultNew.Type = "course"
	var tmpResultNew CourseResult
	var allCoursesNew []Course
	allCoursesNew = nil
	var cstr string = GetCourse(UserName, PassWord)
	err := json.Unmarshal([]byte(cstr), &tmpResultNew)
	if err != nil {
		fmt.Println(err)
	}
	allCoursesNew = tmpResultNew.Data
	for _, c1 := range allCoursesNew {
		schoolWeek := strconv.Itoa(WeekDay)
		intWeek, _ := strconv.Atoi(schoolWeek)
		if c1.Weeks[intWeek] == '1' {
			arr := GetTeacherObj()
			for _, thisteacher := range arr {
				if strings.Contains(c1.CourseID, thisteacher.CourseID) {
					tmpCourseNew.TeacherName = thisteacher.CourseTeacher
					tmpCourseNew.CourseName = thisteacher.CourseName
					tmpCourseNew.RoomName = c1.RoomName
					tmpCourseNew.CourseTimes = c1.CourseTimes
					myWeekCourseNew = append(myWeekCourseNew, tmpCourseNew)
				}
			}
		}
	}
	myCourseResultNew.Data = myWeekCourseNew
	js, _ := json.MarshalIndent(myCourseResultNew, "", "\t")
	return B2S(js)
}

