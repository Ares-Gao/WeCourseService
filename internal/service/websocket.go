package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type userlogin struct {
	Type     string
	UserName string
	PassWord string
	Week     int
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
		for {
			msgType, msg, _ := conn.ReadMessage()
			var u userlogin
			json.Unmarshal([]byte(msg), &u)
			if u.Type == "allcourse" {
				var cstr string = GetCourse(u.UserName, u.PassWord)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "daycourse" {
				var cstr string = GetDayCourse(u.UserName, u.PassWord)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "course" {
				var cstr string = GetWeekCourse(u.UserName, u.PassWord, u.Week)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "weekcourse" {
				var cstr string = GetWeekCourseNew(u.UserName, u.PassWord, u.Week)
				_ = conn.WriteMessage(msgType, []byte(cstr))
			}
			if u.Type == "account" {
				_ = conn.WriteMessage(msgType, []byte(GetAccount(u.UserName, u.PassWord)))
			}
			if u.Type == "login" {
				_ = conn.WriteMessage(msgType, []byte(GetUserLogin(u.UserName, u.PassWord)))
			}
			if u.Type == "week" {
				_ = conn.WriteMessage(msgType, []byte(GetWeekTime(conf.CalendarFirst)))
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

