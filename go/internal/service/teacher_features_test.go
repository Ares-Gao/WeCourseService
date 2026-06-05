package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTeacherCourseActivities(t *testing.T) {
	htmlText := `activity = new TaskActivity(actTeacherId.toString(),"","953174(22101301.16)","工程化学"+periodInfo+"(22101301.16)","690","801楼阶一(北区)","000111",null,null,assistantName,"","");
index =1*unitCount+2;
table0.activities[index][table0.activities[index].length]=activity;
index =1*unitCount+3;
table0.activities[index][table0.activities[index].length]=activity;`
	courses := parseCourseActivities(htmlText)
	if len(courses) != 1 {
		t.Fatalf("expected one course, got %d", len(courses))
	}
	if courses[0].CourseName != "工程化学" || courses[0].RoomName != "801楼阶一(北区)" || len(courses[0].CourseTimes) != 2 {
		t.Fatalf("unexpected course: %+v", courses[0])
	}
}

func TestParseTeacherExamHTML(t *testing.T) {
	htmlText := `<div id="toolbar1"></div><script>bar = bg.ui.toolbar("toolbar1",'我的考试');</script><table><tbody><tr><td>22101301.16</td><td>工程化学</td><td>化学与环境科学学院</td><td>2</td><td>56</td><td><font>时间未发布</font></td><td><font>地点未发布</font></td></tr></tbody></table>`
	exams := ParseTeacherExamHTML(htmlText)
	if len(exams) != 1 {
		t.Fatalf("expected one exam, got %d", len(exams))
	}
	if exams[0].Category != "我的考试" || exams[0].CourseID != "22101301.16" || exams[0].ExamTime != "时间未发布" {
		t.Fatalf("unexpected exam: %+v", exams[0])
	}
}

func TestParseFreeRoomHTML(t *testing.T) {
	htmlText := `<tbody><tr><td>1</td><td>50404</td><td>5号楼</td><td>南区</td><td>琴房</td><td>10</td></tr></tbody>`
	rooms := ParseFreeRoomHTML(htmlText)
	if len(rooms) != 1 {
		t.Fatalf("expected one room, got %d", len(rooms))
	}
	if rooms[0].Name != "50404" || rooms[0].Capacity != "10" {
		t.Fatalf("unexpected room: %+v", rooms[0])
	}
}

func TestParsersWithProbeFiles(t *testing.T) {
	probe := filepath.Join(os.TempDir(), "wecourse_teacher_probe", "teacher_exam_activities.html")
	if _, err := os.Stat(probe); err != nil {
		t.Skip("probe file not available")
	}
	content, err := os.ReadFile(probe)
	if err != nil {
		t.Fatal(err)
	}
	if len(ParseTeacherExamHTML(string(content))) == 0 {
		t.Fatal("expected probe teacher exams")
	}
}
