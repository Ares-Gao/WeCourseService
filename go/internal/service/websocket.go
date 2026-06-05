package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type userlogin struct {
	Type          string
	UserName      string
	PassWord      string
	Week          int
	LoginType     string
	AuthServerURL string
	ExamBatchID   string
	DateBegin     string
	DateEnd       string
	TimeBegin     string
	TimeEnd       string
	RoomTimeType  string
	CycleCount    string
	CycleType     string
	ClassroomType string
	CampusID      string
	BuildingID    string
	Seats         string
	ClassroomName string
}

var build string = "202011211630-Fixed"
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartWebSocket() {
	fmt.Println("Websocket服务开始运行")
	fmt.Println("固件版本：" + build)
	conf := ReadConfig()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		var session userlogin
		for {
			msgType, msg, _ := conn.ReadMessage()
			var u userlogin
			json.Unmarshal([]byte(msg), &u)
			if u.UserName == "" {
				u.UserName = session.UserName
			}
			if u.PassWord == "" {
				u.PassWord = session.PassWord
			}
			if u.LoginType == "" {
				u.LoginType = session.LoginType
			}
			if u.AuthServerURL == "" {
				u.AuthServerURL = session.AuthServerURL
			}
			if u.UserName != "" {
				session.UserName = u.UserName
			}
			if u.PassWord != "" {
				session.PassWord = u.PassWord
			}
			if u.LoginType != "" {
				session.LoginType = u.LoginType
			}
			if u.AuthServerURL != "" {
				session.AuthServerURL = u.AuthServerURL
			}
			if u.Type == "allcourse" {
				var cstr string = GetCourseWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "daycourse" {
				var cstr string = GetDayCourseWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "course" {
				var cstr string = GetWeekCourseWithLogin(u.UserName, u.PassWord, u.Week, u.LoginType, u.AuthServerURL)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "weekcourse" {
				var cstr string = GetWeekCourseNewWithLogin(u.UserName, u.PassWord, u.Week, u.LoginType, u.AuthServerURL)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "ics" {
				_ = conn.WriteMessage(msgType, []byte(GetIcsWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)))
			}
			if u.Type == "account" {
				_ = conn.WriteMessage(msgType, []byte(GetAccount(u.UserName, u.PassWord)))
			}
			if u.Type == "login" {
				_ = conn.WriteMessage(msgType, []byte(GetUserLoginWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)))
			}
			if u.Type == "identity" {
				_ = conn.WriteMessage(msgType, []byte(GetIdentityWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)))
			}
			if u.Type == "teachercourse" {
				_ = conn.WriteMessage(msgType, []byte(GetTeacherCourseWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)))
			}
			if u.Type == "teacherexam" {
				_ = conn.WriteMessage(msgType, []byte(GetTeacherExamWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL, u.ExamBatchID)))
			}
			if u.Type == "teacherexambatch" {
				_ = conn.WriteMessage(msgType, []byte(GetTeacherExamBatchesWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)))
			}
			if u.Type == "freeroom" {
				query := FreeRoomQuery{
					DateBegin: u.DateBegin, DateEnd: u.DateEnd, TimeBegin: u.TimeBegin, TimeEnd: u.TimeEnd,
					RoomApplyTimeType: u.RoomTimeType, CycleCount: u.CycleCount, CycleType: u.CycleType,
					ClassroomTypeID: u.ClassroomType, CampusID: u.CampusID, BuildingID: u.BuildingID,
					Seats: u.Seats, ClassroomName: u.ClassroomName,
				}
				_ = conn.WriteMessage(msgType, []byte(GetFreeRoom(query)))
			}
			if u.Type == "week" {
				_ = conn.WriteMessage(msgType, []byte(GetWeekTime(conf.CalendarFirst)))
			}
			if u.Type == "semester" {
				_ = conn.WriteMessage(msgType, []byte(GetSemesterWithLogin(u.UserName, u.PassWord, u.LoginType, u.AuthServerURL)))
			}
			if u.Type == "teacher" {
				_ = conn.WriteMessage(msgType, []byte(GetTeacher(u.UserName, u.PassWord)))
			}
			if u.Type == "photo" {
				_ = conn.WriteMessage(msgType, []byte(GetPhoto(u.UserName, u.PassWord)))
			}
			if u.Type == "grade" {
				_ = conn.WriteMessage(msgType, []byte(GetGrade(u.UserName, u.PassWord)))
			}
		}

	})
	http.ListenAndServe(":"+strconv.Itoa(conf.SocketPort), nil)
}

func checkErr(err error) {
	if err != nil {
	}
}
