package service

/*
为对付uni-app及微信小程序只能创建一个websocket链接的弱智举动，特针对返回JSON进行统一格式管理
由于Golang没有object，临时想出这个办法管理返回
*/
type WeekCourseResultNew struct {
	Type string
	Data []WeekCourseNew
}
type WeekCourseResult struct {
	Type string
	Data []WeekCourse
}
type CourseResult struct {
	Type string
	Data []Course
}
type SemesterStruct struct {
	SemesterID string
	Ids        string
	Current    bool
}
type SemesterResult struct {
	Type string
	Data []SemesterStruct
}
type DayCourseResult struct {
	Type string
	Data []DayCourse
}
type TeacherResult struct {
	Type string
	Data []TeacherStruct
}
type PhotoResult struct {
	Type string
	Data string
}
type AccountResult struct {
	Type string
	Data StudentStruct
}
type LoginResult struct {
	Type string
	Data string
}
type TimeResult struct {
	Type string
	Data string
}
type GradeResult struct {
	Type string
	Data []GradeStruct
}

