package main

import (
	"fmt"
	"strconv"

	"github.com/Ares-Gao/WeCourseService/go/internal/service"
)

func main() {
	conf := service.ReadConfig()
	fmt.Println("学校名称：" + conf.SchoolName)
	switch conf.MangerType {
	case "supwisdom":
		fmt.Println("教务系统：树维教务系统")
	}
	fmt.Println("绑定端口：" + strconv.Itoa(conf.SocketPort))
	service.StartWebSocket()
}
