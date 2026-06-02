# WeCourseService

WeCourseService 是微课表服务端，基于 [getMyCourses](https://github.com/whoisnian/getMyCourses) 项目改造而来。项目提供基于 WebSocket 的树维教务系统数据查询能力，可用于课表、成绩、教师列表、学籍信息等场景。

项目地址：[Ares-Gao/WeCourseService](https://github.com/Ares-Gao/WeCourseService)

> 当前仅支持树维教务系统（`supwisdom`）。

## 功能

- 验证教务系统账号登录
- 获取当前教学周
- 获取本学期课表、指定周课表、当天课表
- 获取教师列表
- 获取学籍信息
- 获取学籍照片
- 获取成绩
- 使用 GoCache 缓存课表结果，减少对教务系统的频繁访问

## 项目结构

```text
.
├── cmd/
│   └── wecourse-service/     # 服务入口
├── internal/
│   └── service/              # WebSocket、配置读取和教务系统业务逻辑
├── config.json               # 运行配置
├── go.mod                    # Go Module
├── LICENSE
└── README.md
```

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/Ares-Gao/WeCourseService.git
cd WeCourseService
```

### 2. 修改配置

编辑根目录下的 `config.json`：

```json
{
	"SchoolName": "山东商业职业技术学院",
	"MangerType": "supwisdom",
	"MangerURL": "http://szyjxgl.sict.edu.cn:9000/",
	"CalendarFirst": "2020-08-24",
	"SocketPort": 25565
}
```

字段说明：

- `SchoolName`：学校名称
- `MangerType`：教务系统类型，目前仅支持 `supwisdom`
- `MangerURL`：教务系统根地址，不要包含 `eams`
- `CalendarFirst`：校历第一周的星期一，格式为 `YYYY-MM-DD`
- `SocketPort`：WebSocket 服务监听端口

### 3. 启动服务

```bash
go run ./cmd/wecourse-service
```

构建可执行文件：

```bash
go build -o bin/wecourse-service ./cmd/wecourse-service
```

部署时请开放 `SocketPort` 配置的端口。若用于微信小程序，建议通过 Nginx 反向代理并启用 WebSocket over TLS。

## WebSocket 协议

服务通过 WebSocket 通信，消息格式为 JSON。

通用请求示例：

```json
{
	"Type": "course",
	"UserName": "201808830303",
	"PassWord": "7355608",
	"Week": 1
}
```

通用返回格式：

```json
{
	"Type": "course",
	"Data": {}
}
```

`Type` 与请求中的 `Type` 保持一致，`Data` 为接口返回数据。

## 接口

### 登录验证

请求：

```json
{
	"Type": "login",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回：`登录成功` 或 `登录失败`。

### 获取当前教学周

请求：

```json
{
	"Type": "week"
}
```

返回：当前教学周数字。

### 获取教师列表

请求：

```json
{
	"Type": "teacher",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```json
[
	{
		"CourseID": "A000032-.5.11",
		"CourseName": "就业指导实务",
		"CourseCredit": "0.5",
		"CourseTeacher": "王滢"
	}
]
```

### 获取学籍信息

请求：

```json
{
	"Type": "account",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```json
{
	"FullName": "高峰",
	"EnglishName": "Gao Feng",
	"Sex": "男",
	"StartTime": "2018-09-01",
	"EndTime": "2021-06-30",
	"SchoolYear": "3",
	"Type": "专科(普通全日制)",
	"System": "信息与艺术学院(系)",
	"Specialty": "软件技术对口",
	"Class": "软件1803"
}
```

请遵守个人信息保护相关法律法规。无明确业务需要时，不要存储或缓存用户个人信息。

### 获取课程表

请求：

```json
{
	"Type": "course",
	"UserName": "201808830303",
	"PassWord": "7355608",
	"Week": 1
}
```

说明：

- `Week` 为 `0` 时，返回本学期完整课表
- `Week` 大于 `0` 时，返回指定教学周课表

指定周课表返回示例：

```json
[
	{
		"CourseName": "JavaScript程序设计",
		"TeacherName": "薛现伟",
		"RoomName": "301,计算机基础实训室(一)",
		"DayOfTheWeek": 3,
		"TimeOfTheDay": "5,6"
	}
]
```

### 获取学籍照片

请求：

```json
{
	"Type": "photo",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回：`data:image/jpg;base64,...` 格式的图片数据。

请遵守个人信息保护和肖像权相关法律法规。

### 获取成绩

请求：

```json
{
	"Type": "grade",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```json
[
	{
		"CourseID": "A000003-4",
		"CourseName": "大学英语（一）A",
		"CourseTerm": "2018-2019 1",
		"CourseCredit": "4",
		"CourseGrade": "64",
		"GradePoint": "1.5"
	}
]
```

## 许可证

本项目使用 MIT License，详见 [LICENSE](./LICENSE)。

项目已获得软件著作权，登记号：`2019SR0620279`。
