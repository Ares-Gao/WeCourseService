package service

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TeacherExam struct {
	Category     string
	CourseID     string
	CourseName   string
	Department   string
	Credit       string
	StudentCount string
	Invigilators string
	ExamTime     string
	ExamRoom     string
}

type TeacherExamBatch struct {
	ExamBatchID string
	Name        string
	Selected    bool
}

type TeacherExamResult struct {
	Type string
	Data []TeacherExam
}

type TeacherExamBatchResult struct {
	Type string
	Data []TeacherExamBatch
}

type FreeRoom struct {
	Index    string
	Name     string
	Building string
	Campus   string
	TypeName string
	Capacity string
}

type FreeRoomQuery struct {
	DateBegin         string
	DateEnd           string
	TimeBegin         string
	TimeEnd           string
	RoomApplyTimeType string
	CycleCount        string
	CycleType         string
	ClassroomTypeID   string
	CampusID          string
	BuildingID        string
	Seats             string
	ClassroomName     string
}

type FreeRoomResult struct {
	Type string
	Data []FreeRoom
}

func GetTeacherCourse(UserName, PassWord string) string {
	return GetTeacherCourseWithLogin(UserName, PassWord, "", "")
}

func GetTeacherCourseWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	result := CourseResult{Type: "teachercourse"}
	conf := ReadConfig()
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	indexHTML, err := httpGetString(client, conf.MangerURL+"eams/courseTableForTeacher.action")
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	ids, semesterID, ok := getTeacherCourseTableParams(indexHTML)
	if !ok {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	form := url.Values{}
	form.Set("ignoreHead", "1")
	form.Set("setting.forSemester", "1")
	form.Set("ids", ids)
	form.Set("setting.kind", "teacher")
	form.Set("semester.id", semesterID)
	tableHTML, err := httpPostFormString(client, conf.MangerURL+"eams/courseTableForTeacher!courseTable.action", form)
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	result.Data = parseCourseActivities(tableHTML)
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}

func GetTeacherExam(UserName, PassWord, examBatchID string) string {
	return GetTeacherExamWithLogin(UserName, PassWord, "", "", examBatchID)
}

func GetTeacherExamWithLogin(UserName, PassWord, loginType, authServerURL, examBatchID string) string {
	result := TeacherExamResult{Type: "teacherexam"}
	conf := ReadConfig()
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	indexHTML, err := httpGetString(client, conf.MangerURL+"eams/teacherExamTable.action")
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	if examBatchID == "" {
		examBatchID = selectedExamBatchID(indexHTML)
	}
	if examBatchID == "" {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	examHTML, err := httpGetString(client, conf.MangerURL+"eams/teacherExamTable!examAtivities.action?examBatch.id="+url.QueryEscape(examBatchID))
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	result.Data = ParseTeacherExamHTML(examHTML)
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}

func GetTeacherExamBatchesWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	result := TeacherExamBatchResult{Type: "teacherexambatch"}
	conf := ReadConfig()
	client, err := CreateLoggedInClient(conf, UserName, PassWord, LoginOptions{LoginType: loginType, AuthServerURL: authServerURL})
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	indexHTML, err := httpGetString(client, conf.MangerURL+"eams/teacherExamTable.action")
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	result.Data = ParseTeacherExamBatches(indexHTML)
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}

func GetFreeRoom(query FreeRoomQuery) string {
	result := FreeRoomResult{Type: "freeroom"}
	conf := ReadConfig()
	if query.DateBegin == "" {
		query.DateBegin = time.Now().Format("2006-01-02")
	}
	if query.DateEnd == "" {
		query.DateEnd = query.DateBegin
	}
	if query.TimeBegin == "" {
		query.TimeBegin = "1"
	}
	if query.TimeEnd == "" {
		query.TimeEnd = query.TimeBegin
	}
	if query.RoomApplyTimeType == "" {
		query.RoomApplyTimeType = "0"
	}
	if query.CycleCount == "" {
		query.CycleCount = "1"
	}
	if query.CycleType == "" {
		query.CycleType = "1"
	}
	form := url.Values{}
	form.Set("classroom.type.id", query.ClassroomTypeID)
	form.Set("classroom.campus.id", query.CampusID)
	form.Set("classroom.building.id", query.BuildingID)
	form.Set("seats", query.Seats)
	form.Set("classroom.name", query.ClassroomName)
	form.Set("cycleTime.cycleCount", query.CycleCount)
	form.Set("cycleTime.cycleType", query.CycleType)
	form.Set("cycleTime.dateBegin", query.DateBegin)
	form.Set("cycleTime.dateEnd", query.DateEnd)
	form.Set("roomApplyTimeType", query.RoomApplyTimeType)
	form.Set("timeBegin", query.TimeBegin)
	form.Set("timeEnd", query.TimeEnd)
	htmlText, err := httpPostFormString(http.DefaultClient, conf.MangerURL+"eams/publicFree!search.action", form)
	if err != nil {
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	result.Data = ParseFreeRoomHTML(htmlText)
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}

func getTeacherCourseTableParams(htmlText string) (string, string, bool) {
	idsReg := regexp.MustCompile(`name=["']ids["'][^>]*value=["']([^"']+)["']`)
	idsMatch := idsReg.FindStringSubmatch(htmlText)
	semesterReg := regexp.MustCompile(`semesterCalendar\(\{[^}]*value:["']([^"']+)["']`)
	semesterMatch := semesterReg.FindStringSubmatch(htmlText)
	if len(idsMatch) < 2 || len(semesterMatch) < 2 {
		return "", "", false
	}
	return idsMatch[1], semesterMatch[1], true
}

func parseCourseActivities(htmlText string) []Course {
	reg1 := regexp.MustCompile(`TaskActivity\(actTeacherId(?:\.toString\(\)|\.join\(','\)),[^,]*,"(.*)","(.*)\(.*\)","(.*)","(.*)","(.*)",null,null,assistantName,"",""\);((?:\s*index =\d+\*unitCount\+\d+;\s*.*\s)+)`)
	reg2 := regexp.MustCompile(`\s*index =(\d+)\*unitCount\+(\d+);\s*`)
	coursesStr := reg1.FindAllStringSubmatch(htmlText, -1)
	courses := make([]Course, 0, len(coursesStr))
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
			match := reg2.FindStringSubmatch(indexStr)
			if len(match) < 3 {
				continue
			}
			day, _ := strconv.Atoi(match[1])
			slot, _ := strconv.Atoi(match[2])
			course.CourseTimes = append(course.CourseTimes, CourseTime{DayOfTheWeek: day, TimeOfTheDay: slot})
		}
		courses = append(courses, course)
	}
	return courses
}

func ParseTeacherExamHTML(htmlText string) []TeacherExam {
	sections := splitGridSections(htmlText)
	exams := []TeacherExam{}
	for _, section := range sections {
		category := sectionTitle(section)
		for _, cells := range parseTableCells(section) {
			if len(cells) < 7 || cells[0] == "" {
				continue
			}
			exam := TeacherExam{Category: category, CourseID: cells[0], CourseName: cells[1], Department: cells[2], Credit: cells[3]}
			if len(cells) >= 8 {
				exam.Invigilators = cells[4]
				exam.StudentCount = cells[5]
				exam.ExamTime = cells[6]
				exam.ExamRoom = cells[7]
			} else {
				exam.StudentCount = cells[4]
				exam.ExamTime = cells[5]
				exam.ExamRoom = cells[6]
			}
			exams = append(exams, exam)
		}
	}
	return exams
}

func ParseFreeRoomHTML(htmlText string) []FreeRoom {
	rooms := []FreeRoom{}
	for _, cells := range parseTableCells(htmlText) {
		if len(cells) < 6 || cells[0] == "" {
			continue
		}
		rooms = append(rooms, FreeRoom{
			Index: cells[0], Name: cells[1], Building: cells[2],
			Campus: cells[3], TypeName: cells[4], Capacity: cells[5],
		})
	}
	return rooms
}

func parseTableCells(htmlText string) [][]string {
	rowReg := regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	cellReg := regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	rows := [][]string{}
	for _, row := range rowReg.FindAllStringSubmatch(htmlText, -1) {
		matches := cellReg.FindAllStringSubmatch(row[1], -1)
		if len(matches) == 0 {
			continue
		}
		cells := make([]string, 0, len(matches))
		for _, match := range matches {
			cells = append(cells, cleanHTMLCell(match[1]))
		}
		rows = append(rows, cells)
	}
	return rows
}

func cleanHTMLCell(value string) string {
	tagReg := regexp.MustCompile(`(?is)<[^>]+>`)
	value = tagReg.ReplaceAllString(value, "")
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "\u00a0", " ")
	return strings.Join(strings.Fields(value), " ")
}

func splitGridSections(htmlText string) []string {
	toolbarReg := regexp.MustCompile(`(?is)<div id="toolbar[^"]*"`)
	matches := toolbarReg.FindAllStringIndex(htmlText, -1)
	if len(matches) == 0 {
		return []string{htmlText}
	}
	sections := make([]string, 0, len(matches))
	for i, match := range matches {
		start := match[0]
		end := len(htmlText)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		sections = append(sections, htmlText[start:end])
	}
	return sections
}

func sectionTitle(section string) string {
	titleReg := regexp.MustCompile(`bg\.ui\.toolbar\("[^"]+",'([^']*)'`)
	match := titleReg.FindStringSubmatch(section)
	if len(match) >= 2 {
		return cleanHTMLCell(match[1])
	}
	return ""
}

func ParseTeacherExamBatches(htmlText string) []TeacherExamBatch {
	reg := regexp.MustCompile(`(?is)<option\s+value=["']([^"']+)["']([^>]*)>(.*?)</option>`)
	matches := reg.FindAllStringSubmatch(htmlText, -1)
	batches := make([]TeacherExamBatch, 0, len(matches))
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		batches = append(batches, TeacherExamBatch{
			ExamBatchID: match[1],
			Name:        cleanHTMLCell(match[3]),
			Selected:    strings.Contains(strings.ToLower(match[2]), "selected"),
		})
	}
	return batches
}

func selectedExamBatchID(htmlText string) string {
	for _, batch := range ParseTeacherExamBatches(htmlText) {
		if batch.Selected {
			return batch.ExamBatchID
		}
	}
	reg := regexp.MustCompile(`<option value=["']([^"']+)["'][^>]*selected`)
	match := reg.FindStringSubmatch(htmlText)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

func httpGetString(client *http.Client, target string) (string, error) {
	resp, err := client.Get(target)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	return string(content), err
}

func httpPostFormString(client *http.Client, target string, form url.Values) (string, error) {
	resp, err := client.PostForm(target, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	return string(content), err
}
