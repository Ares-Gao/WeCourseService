package service

import (
	"strings"
	"testing"
)

func TestGenerateCourseICSUsesConfiguredClassTimeSlots(t *testing.T) {
	conf := Config{
		SchoolName:       "测试学校",
		CalendarFirst:    "2026-03-02",
		CalendarTimezone: "Asia/Shanghai",
		CalendarName:     "测试课表",
		ClassTimeSlots: []ClassTimeSlot{
			{Start: "08:10", End: "08:55"},
			{Start: "09:05", End: "09:50"},
		},
	}
	courses := []Course{
		{
			CourseID:   "C001",
			CourseName: "测试课程",
			RoomID:     "R001",
			RoomName:   "一号教室",
			Weeks:      "010",
			CourseTimes: []CourseTime{
				{DayOfTheWeek: 2, TimeOfTheDay: 0},
				{DayOfTheWeek: 2, TimeOfTheDay: 1},
			},
		},
	}

	ics := GenerateCourseICS(conf, courses)
	if !strings.Contains(ics, "DTSTART;TZID=Asia/Shanghai:20260304T081000") {
		t.Fatalf("ics did not use configured start time: %s", ics)
	}
	if !strings.Contains(ics, "DTEND;TZID=Asia/Shanghai:20260304T095000") {
		t.Fatalf("ics did not use configured end time: %s", ics)
	}
	if !strings.Contains(ics, "SUMMARY:测试课程") {
		t.Fatalf("ics summary missing: %s", ics)
	}
}
