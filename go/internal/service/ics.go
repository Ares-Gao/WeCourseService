package service

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type IcsResult struct {
	Type string
	Data string
}

func GetIcs(UserName, PassWord string) string {
	return GetIcsWithLogin(UserName, PassWord, "", "")
}

func GetIcsWithLogin(UserName, PassWord, loginType, authServerURL string) string {
	result := IcsResult{Type: "ics"}
	var courseResult CourseResult
	courseJSON := GetCourseWithLogin(UserName, PassWord, loginType, authServerURL)
	if err := json.Unmarshal([]byte(courseJSON), &courseResult); err != nil {
		result.Data = ""
		js, _ := json.MarshalIndent(result, "", "\t")
		return B2S(js)
	}
	result.Data = GenerateCourseICS(ReadConfig(), courseResult.Data)
	js, _ := json.MarshalIndent(result, "", "\t")
	return B2S(js)
}

func GenerateCourseICS(conf Config, courses []Course) string {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	timezone := valueOr(conf.CalendarTimezone, "Asia/Shanghai")
	calendarName := valueOr(conf.CalendarName, conf.SchoolName+"课表")
	slots := classTimeSlots(conf)
	firstMonday, err := time.ParseInLocation("2006-01-02", conf.CalendarFirst, location)
	if err != nil {
		firstMonday = time.Now().In(location)
	}

	builder := strings.Builder{}
	builder.WriteString("BEGIN:VCALENDAR\r\n")
	builder.WriteString("VERSION:2.0\r\n")
	builder.WriteString("PRODID:-//Ares-Gao//WeCourseService//CN\r\n")
	builder.WriteString("CALSCALE:GREGORIAN\r\n")
	builder.WriteString("METHOD:PUBLISH\r\n")
	builder.WriteString(foldICalLine("X-WR-CALNAME:" + escapeICalText(calendarName)))
	builder.WriteString(foldICalLine("X-WR-TIMEZONE:" + timezone))

	now := time.Now().UTC().Format("20060102T150405Z")
	for _, course := range courses {
		dayTimes := groupCourseTimes(course.CourseTimes)
		for weekIndex, enabled := range course.Weeks {
			if weekIndex == 0 || enabled != '1' {
				continue
			}
			for day, times := range dayTimes {
				startSlot := times[0]
				endSlot := times[len(times)-1]
				if startSlot < 0 || startSlot >= len(slots) || endSlot < 0 || endSlot >= len(slots) {
					continue
				}
				date := firstMonday.AddDate(0, 0, (weekIndex-1)*7+day)
				startAt := combineDateTime(date, slots[startSlot].Start, location)
				endAt := combineDateTime(date, slots[endSlot].End, location)
				uid := uidForCourse(course, weekIndex, day, startSlot)

				builder.WriteString("BEGIN:VEVENT\r\n")
				builder.WriteString(foldICalLine("UID:" + uid))
				builder.WriteString("DTSTAMP:" + now + "\r\n")
				builder.WriteString(foldICalLine("DTSTART;TZID=" + timezone + ":" + startAt.Format("20060102T150405")))
				builder.WriteString(foldICalLine("DTEND;TZID=" + timezone + ":" + endAt.Format("20060102T150405")))
				builder.WriteString(foldICalLine("SUMMARY:" + escapeICalText(course.CourseName)))
				if course.RoomName != "" {
					builder.WriteString(foldICalLine("LOCATION:" + escapeICalText(course.RoomName)))
				}
				description := "CourseID: " + course.CourseID + "\nRoomID: " + course.RoomID
				builder.WriteString(foldICalLine("DESCRIPTION:" + escapeICalText(description)))
				builder.WriteString("END:VEVENT\r\n")
			}
		}
	}
	builder.WriteString("END:VCALENDAR\r\n")
	return builder.String()
}

func classTimeSlots(conf Config) []ClassTimeSlot {
	if len(conf.ClassTimeSlots) > 0 {
		return conf.ClassTimeSlots
	}
	return []ClassTimeSlot{
		{Start: "08:00", End: "08:45"},
		{Start: "08:55", End: "09:40"},
		{Start: "10:00", End: "10:45"},
		{Start: "10:55", End: "11:40"},
		{Start: "14:00", End: "14:45"},
		{Start: "14:55", End: "15:40"},
		{Start: "16:00", End: "16:45"},
		{Start: "16:55", End: "17:40"},
		{Start: "19:00", End: "19:45"},
		{Start: "19:55", End: "20:40"},
		{Start: "20:50", End: "21:35"},
		{Start: "21:45", End: "22:30"},
	}
}

func groupCourseTimes(courseTimes []CourseTime) map[int][]int {
	result := map[int][]int{}
	for _, item := range courseTimes {
		result[item.DayOfTheWeek] = append(result[item.DayOfTheWeek], item.TimeOfTheDay)
	}
	for day := range result {
		sort.Ints(result[day])
	}
	return result
}

func combineDateTime(date time.Time, clock string, location *time.Location) time.Time {
	parts := strings.Split(clock, ":")
	hour, minute := 0, 0
	if len(parts) >= 2 {
		hour, _ = strconv.Atoi(parts[0])
		minute, _ = strconv.Atoi(parts[1])
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, location)
}

func uidForCourse(course Course, weekIndex, day, startSlot int) string {
	raw := course.CourseID + "-" + strconv.Itoa(weekIndex) + "-" + strconv.Itoa(day) + "-" + strconv.Itoa(startSlot)
	clean := regexp.MustCompile(`[^0-9A-Za-z_-]`).ReplaceAllString(raw, "-")
	return clean + "@wecourse.service"
}

func escapeICalText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, ";", `\;`)
	value = strings.ReplaceAll(value, ",", `\,`)
	return value
}

func foldICalLine(line string) string {
	runes := []rune(line)
	if len(runes) <= 75 {
		return line + "\r\n"
	}
	builder := strings.Builder{}
	for len(runes) > 75 {
		builder.WriteString(string(runes[:75]) + "\r\n ")
		runes = runes[75:]
	}
	builder.WriteString(string(runes) + "\r\n")
	return builder.String()
}
