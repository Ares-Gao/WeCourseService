# WeCourseService

WeCourseService 是面向教务系统数据查询的多语言 SDK 与服务端集合，基于 [getMyCourses](https://github.com/whoisnian/getMyCourses) 项目改造而来。项目提供树维教务系统的数据查询能力，可用于课表、成绩、教师列表、学籍信息等场景。

项目地址：[Ares-Gao/WeCourseService](https://github.com/Ares-Gao/WeCourseService)

> 当前仅支持树维教务系统（`supwisdom`）。

## 6 Years Ago 特殊纪念更新

本次更新是项目发布 6 年后的一次特殊纪念整理：仓库重新整理为多语言 SDK 形态，并从单一 WebSocket 服务逐步扩展为可按需选择的服务端、库和协议实现。

本次重写与整理工作由 Codex && ChatGPT 5.5 配合完成。

## 项目目标

WeCourseService 不再只定位为某一种语言的 WebSocket 服务端。后续会尽可能支持全部主流编程语言，并提供更接近 SDK 的使用方式，让使用者可以按自己的项目选择语言、接入方式和部署形态。

计划支持的形态包括：

- WebSocket 服务
- HTTP API 服务
- 语言原生 SDK
- 命令行工具
- 可嵌入的业务模块

## 已支持语言

| 版本 | 目录 | 状态 | 适合场景 |
| --- | --- | --- | --- |
| Go | [go](./go) | 已支持 | 需要单文件构建、低运行时依赖、服务端长期运行 |
| Python | [python](./python) | 已支持 | 需要脚本化部署、快速二次开发、已有 Python 环境 |
| C# | [csharp](./csharp) | 已支持 | 需要接入 .NET、桌面程序、ASP.NET 服务 |
| PHP | [php](./php) | 已支持 | 需要接入传统 Web 项目、Laravel/Symfony 或轻量脚本 |
| Java | [java](./java) | 已支持 | 需要接入 Spring、JVM 服务或企业后端 |

后续计划会继续整理 JavaScript/TypeScript、Rust、Kotlin、Swift 等主流语言版本。具体实现会按稳定性逐步加入仓库。

## 功能

- 验证教务系统账号登录
- 获取当前教学周
- 查询当前学期 ID 与课表请求参数，方便用户切换学期或排查空课表
- 获取本学期课表、指定周课表、当天课表
- 生成可导入日历软件的 ICS 课表文件内容
- 获取教师列表
- 获取学籍信息
- 获取学籍照片
- 获取成绩
- 缓存课表结果，减少对教务系统的频繁访问
- 逐步提供多语言 SDK 与多种接入方式

本次整理已修复历史版本中写死 `semester.id=30` 的问题。现在课表和教师查询会从教务系统页面动态解析当前 `semester.id` 与 `ids`，避免不同学校、不同学期因为固定学期 ID 导致无法获取课表。

## 项目结构

```text
.
├── go/                      # Go 版本
│   ├── cmd/
│   │   ├── wecourse-service  # WebSocket 服务入口
│   ├── internal/
│   ├── config.json
│   └── go.mod
├── configtool/              # 全语言共用图形化配置工具
├── python/                  # Python 版本
│   ├── wecourse_service/
│   ├── requirements.txt
│   └── README.md
├── csharp/                  # C# 版本
├── php/                     # PHP 版本
├── java/                    # Java 版本
├── config.json              # Python 版本默认读取的根配置
├── LICENSE
└── README.md
```

## 快速开始

### Go 版本

```bash
cd go
go run ./cmd/wecourse-service
```

更多说明见 [go/README.md](./go/README.md)。

### Python 版本

```bash
cd python
python -m venv .venv
.venv\Scripts\activate
pip install -r requirements.txt
python -m wecourse_service
```

更多说明见 [python/README.md](./python/README.md)。

### C# 版本

```bash
cd csharp
dotnet build
```

更多说明见 [csharp/README.md](./csharp/README.md)。

### PHP 版本

更多说明见 [php/README.md](./php/README.md)。

### Java 版本

```bash
cd java
javac -encoding UTF-8 -d out src/main/java/io/github/aresgao/wecourseservice/*.java
```

更多说明见 [java/README.md](./java/README.md)。

## 配置

配置文件示例：

```json
{
	"SchoolName": "山东商业职业技术学院",
	"MangerType": "supwisdom",
	"MangerURL": "http://szyjxgl.sict.edu.cn:9000/",
	"CalendarFirst": "2020-08-24",
	"SocketPort": 25565,
	"LoginType": "direct",
	"AuthServerURL": "",
	"ServiceURL": "",
	"AuthServerAutoCaptcha": true,
	"AuthServerCaptchaRetries": 3,
	"DdddOcrOnnxRuntimeLibPath": "",
	"DdddOcrModelPath": "",
	"DdddOcrDictPath": "",
	"DdddOcrDetModelPath": "",
	"DdddOcrUseCustomModel": false,
	"CalendarTimezone": "Asia/Shanghai",
	"CalendarName": "微课表",
	"ClassTimeSlots": [
		{"Start": "08:00", "End": "08:45"},
		{"Start": "08:55", "End": "09:40"},
		{"Start": "10:00", "End": "10:45"},
		{"Start": "10:55", "End": "11:40"},
		{"Start": "14:00", "End": "14:45"},
		{"Start": "14:55", "End": "15:40"},
		{"Start": "16:00", "End": "16:45"},
		{"Start": "16:55", "End": "17:40"},
		{"Start": "19:00", "End": "19:45"},
		{"Start": "19:55", "End": "20:40"},
		{"Start": "20:50", "End": "21:35"},
		{"Start": "21:45", "End": "22:30"}
	]
}
```

字段说明：

- `SchoolName`：学校名称
- `MangerType`：教务系统类型，目前仅支持 `supwisdom`
- `MangerURL`：教务系统根地址，不要包含 `eams`
- `CalendarFirst`：校历第一周的星期一，格式为 `YYYY-MM-DD`
- `SocketPort`：WebSocket 服务监听端口
- `LoginType`：默认登录方式，`direct` 为旧版树维直登，`authserver` 为统一身份认证；实际请求可通过同名参数覆盖
- `AuthServerURL`：默认统一身份认证登录地址，实际请求可通过同名参数覆盖
- `ServiceURL`：统一身份认证完成后跳转的教务系统 SSO 地址，通常已包含在 `AuthServerURL` 的 `service` 参数中，可按语言实现需要保留
- `AuthServerAutoCaptcha`：是否自动处理 authserver 普通图片验证码，Python 版本默认使用 `ddddocr`
- `AuthServerCaptchaRetries`：验证码识别失败时的重试次数
- `DdddOcrOnnxRuntimeLibPath`：Go 版本启用 ddddocr 时使用的 `onnxruntime.dll` / `.so` / `.dylib` 路径
- `DdddOcrModelPath`：Go 版本 ddddocr 分类模型路径，例如 `common.onnx`
- `DdddOcrDictPath`：Go 版本 ddddocr 字典文件路径，每行一个字符，需要与模型字符集一致
- `DdddOcrDetModelPath`：Go 版本 ddddocr 检测模型路径，普通字符验证码可留空
- `DdddOcrUseCustomModel`：Go 版本是否使用自训练 ddddocr 模型
- `CalendarTimezone`：ICS 日历使用的时区，默认 `Asia/Shanghai`
- `CalendarName`：ICS 日历名称
- `ClassTimeSlots`：学校真实作息时间表，第 1 节课对应数组第 0 项，生成 ICS 时按这里的开始和结束时间写入事件

Go 版本默认读取 `go/config.json`。Python 版本默认读取仓库根目录的 `config.json`，也可以通过 `WECOURSE_CONFIG` 指定配置文件路径。

## ConfigTool 图形化配置工具

配置项增多后，推荐使用 ConfigTool 维护配置。它是一个本地 Web 图形界面，支持中文/English 切换，会自动定位仓库根目录，读取已有配置；如果 `config.json` 或 `go/config.json` 不存在，会按默认模板生成。

```bash
cd configtool
go run .
```

启动后默认打开 `http://127.0.0.1:9630`。保存时会同步写入：

- `config.json`：Python、C#、PHP、Java 默认共用
- `go/config.json`：Go 服务默认读取

如需指定监听地址：

```bash
go run . -addr 127.0.0.1:9631
```

### Authserver 登录配置示例

部分学校不再使用 `eams/login.action` 直登，而是通过统一身份认证平台登录，再跳转到树维教务系统。例如：

```json
{
	"SchoolName": "陕西理工大学",
	"MangerType": "supwisdom",
	"MangerURL": "http://jwgl.snut.edu.cn/",
	"CalendarFirst": "2024-09-02",
	"SocketPort": 25565,
	"LoginType": "authserver",
	"AuthServerURL": "https://authserver.snut.edu.cn/authserver/login?service=http%3A%2F%2Fjwgl.snut.edu.cn%2Feams%2FssoLogin.action",
	"ServiceURL": "http://jwgl.snut.edu.cn/eams/ssoLogin.action",
	"AuthServerAutoCaptcha": true,
	"AuthServerCaptchaRetries": 3
}
```

`authserver` 模式会读取登录页中的 `execution` 和 `pwdEncryptSalt`，并按页面脚本使用 AES-CBC-PKCS7 加密密码后提交。

当前 authserver 登录分支已接入 Go、Python、C#、PHP、Java SDK。Go 版本已完成统一登录 helper、验证码识别和 SSO 后课表解析链路。

验证码处理：

- Python：已接入 `ddddocr`，可自动识别普通图片验证码。
- Go：已接入 `github.com/getcharzp/go-ocr/ddddocr`，配置 ONNXRuntime 动态库和 ddddocr ONNX 模型后可自动识别普通图片验证码。
- C#：提供 `ICaptchaSolver` 插件口，可接入 Tesseract、PaddleOCR 服务或自建 OCR 服务。
- PHP：构造 `SupwisdomClient` 时可传入 callable 作为验证码识别器，可接入 Tesseract CLI 或外部 OCR 服务。
- Java：提供 `CaptchaSolver` 插件口，可接入 Tess4J、PaddleOCR 服务或自建 OCR 服务。
- 滑块验证码需要单独接缺口识别和轨迹提交，当前先预留为后续增强项。

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

## 返回格式

```json
{
	"Type": "login",
	"Data": {}
}
```

由于 uni-app 只支持同时连接一个 WebSocket，为方便管理，返回格式统一为 `Type + Data`。

`Type` 与请求中的 `Type` 保持一致，`Data` 为接口返回数据。以下接口章节中的“返回示例”均代表完整返回结果中的 `Data` 内容。

登录方式不再由 `config.json` 强制二选一。`config.json` 只提供默认值；每次请求都可以传入 `LoginType` 决定走树维直登还是 authserver，必要时也可以传入 `AuthServerURL` 覆盖默认统一认证入口。

请求级 authserver 示例：

```json
{
	"Type": "allcourse",
	"UserName": "201808830303",
	"PassWord": "7355608",
	"LoginType": "authserver",
	"AuthServerURL": "https://authserver.snut.edu.cn/authserver/login?service=http%3A%2F%2Fjwgl.snut.edu.cn%2Feams%2FssoLogin.action"
}
```

## 接口

### 登录验证

该功能用于验证账号密码是否能成功登录教务系统。

```json
{
	"Type": "login",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```json
"登录成功"
```

或：

```json
"登录失败"
```

### 获取当前教学周

这个功能主要是为了方便同学们获取现在是第几周。

```json
{
	"Type": "week"
}
```

返回示例：

```json
"8"
```

### 识别账号身份

登录后读取 `eams/homeExt.action`，优先根据 `security.userCategoryId` 判断账号身份：`1` 为学生，`2` 为教师；若该 hidden input 不存在，会继续用学生/教师菜单 URL 作为兜底判断。

```json
{
	"Type": "identity",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```json
{
	"Type": "identity",
	"Data": {
		"Role": "student",
		"RoleName": "学生",
		"UserCategoryID": "1"
	}
}
```

### 查询学期 ID

```json
{
	"Type": "semester",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```json
{
	"Type": "semester",
	"Data": [
		{
			"SemesterID": "142",
			"Ids": "126112",
			"Current": true
		}
	]
}
```

`SemesterID` 对应树维接口中的 `semester.id`，`Ids` 为当前学生课表查询所需的内部参数。当前版本优先返回教务页面上识别到的当前学期参数，便于使用者确认或在后续调用中扩展指定学期能力。

### 获取教师列表

这个功能用于获取本学期课程对应的教师列表。

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
	},
	{
		"CourseID": "A005217-2.06",
		"CourseName": "社会心理学",
		"CourseCredit": "2",
		"CourseTeacher": "刘悦"
	},
	{
		"CourseID": "A080311-4.11",
		"CourseName": "JavaScript程序设计",
		"CourseCredit": "4",
		"CourseTeacher": "薛现伟"
	},
	{
		"CourseID": "A080910-6.09",
		"CourseName": "HTML5混合App开发",
		"CourseCredit": "6",
		"CourseTeacher": "王永乾"
	},
	{
		"CourseID": "A080913-6.09",
		"CourseName": "PHP动态网站开发",
		"CourseCredit": "6",
		"CourseTeacher": "郑春光"
	}
]
```

`CourseID` 为教务系统内部分配的课程 ID，`CourseName` 为课程名称，`CourseCredit` 为课程学分，`CourseTeacher` 为任课教师。

### 获取学籍信息

这个功能可以用来做用户身份识别，比如展示资料卡、电子学籍卡等。

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

`FullName` 为中文全名，`EnglishName` 为英文名称，`Sex` 为性别，`StartTime` 为入学时间，`EndTime` 为毕业时间，`SchoolYear` 为学年，`Type` 为学历类型，`System` 为院系，`Specialty` 为专业，`Class` 为班级。

请遵守个人信息保护相关法律法规。无明确业务需要时，不要存储或缓存用户个人信息，严禁将用户个人信息用于违法犯罪活动。

### 获取课程表

本程序的核心功能，用于获取本人的本学期全部课程表或指定周课程表。

```json
{
	"Type": "course",
	"UserName": "201808830303",
	"PassWord": "7355608",
	"Week": 1
}
```

- `Week` 为 `0` 时，返回本学期完整课表
- `Week` 大于 `0` 时，返回指定教学周课表

完整课表返回示例：

```json
[
	{
		"CourseID": "14290(A080311-4.11)",
		"CourseName": "JavaScript程序设计",
		"RoomID": "1526",
		"RoomName": "301,计算机基础实训室(一)",
		"Weeks": "01111111111111111110000000000000000000000000000000000",
		"CourseTimes": [
			{
				"DayOfTheWeek": 3,
				"TimeOfTheDay": 4
			},
			{
				"DayOfTheWeek": 3,
				"TimeOfTheDay": 5
			}
		]
	},
	{
		"CourseID": "14290(A080311-4.11)",
		"CourseName": "JavaScript程序设计",
		"RoomID": "1556",
		"RoomName": "311,影视多媒体实训室",
		"Weeks": "01111111111111111110000000000000000000000000000000000",
		"CourseTimes": [
			{
				"DayOfTheWeek": 1,
				"TimeOfTheDay": 2
			},
			{
				"DayOfTheWeek": 1,
				"TimeOfTheDay": 3
			}
		]
	},
	{
		"CourseID": "19827(A080910-6.09)",
		"CourseName": "HTML5混合App开发",
		"RoomID": "-1",
		"RoomName": "停课",
		"Weeks": "00000000000010000000000000000000000000000000000000000",
		"CourseTimes": [
			{
				"DayOfTheWeek": 2,
				"TimeOfTheDay": 0
			},
			{
				"DayOfTheWeek": 2,
				"TimeOfTheDay": 1
			}
		]
	},
	{
		"CourseID": "19803(A080913-6.09)",
		"CourseName": "PHP动态网站开发",
		"RoomID": "1558",
		"RoomName": "317,信息决策实训室j",
		"Weeks": "01111111111111011110000000000000000000000000000000000",
		"CourseTimes": [
			{
				"DayOfTheWeek": 3,
				"TimeOfTheDay": 2
			},
			{
				"DayOfTheWeek": 3,
				"TimeOfTheDay": 3
			}
		]
	},
	{
		"CourseID": "8892(A000032-.5.11)",
		"CourseName": "就业指导实务",
		"RoomID": "1487",
		"RoomName": "本部E206",
		"Weeks": "01111000000000000000000000000000000000000000000000000",
		"CourseTimes": [
			{
				"DayOfTheWeek": 2,
				"TimeOfTheDay": 2
			},
			{
				"DayOfTheWeek": 2,
				"TimeOfTheDay": 3
			}
		]
	}
]
```

`CourseID` 为教务系统分配的课程 ID，数字 ID 括号内的文本与教师列表一致；`CourseName` 为课程名称；`RoomID` 为教务系统内部教室 ID；`RoomName` 为教室名称；`Weeks` 为上课周，树维周次字符串第 0 位通常为占位，第 1 位对应第一周，有课为 `1`，无课为 `0`；`CourseTimes` 为课程上课时间，`DayOfTheWeek` 表示周几上课，`0` 表示周一，`TimeOfTheDay` 表示当天第几节，`0` 表示第一节。

指定周课程表返回示例：

```json
[
	{
		"CourseName": "JavaScript程序设计",
		"TeacherName": "薛现伟",
		"RoomName": "301,计算机基础实训室(一)",
		"DayOfTheWeek": 3,
		"TimeOfTheDay": "5,6"
	},
	{
		"CourseName": "JavaScript程序设计",
		"TeacherName": "薛现伟",
		"RoomName": "311,影视多媒体实训室",
		"DayOfTheWeek": 1,
		"TimeOfTheDay": "3,4"
	},
	{
		"CourseName": "HTML5混合App开发",
		"TeacherName": "王永乾",
		"RoomName": "317,信息决策实训室j",
		"DayOfTheWeek": 3,
		"TimeOfTheDay": "1,2"
	},
	{
		"CourseName": "PHP动态网站开发",
		"TeacherName": "郑春光",
		"RoomName": "304小,计算机基础实训室(三)",
		"DayOfTheWeek": 4,
		"TimeOfTheDay": "5,6"
	},
	{
		"CourseName": "就业指导实务",
		"TeacherName": "王滢",
		"RoomName": "本部E206",
		"DayOfTheWeek": 2,
		"TimeOfTheDay": "3,4"
	}
]
```

指定周课程表会自动整合同一天的节次。`CourseName` 为课程名称，`TeacherName` 为教师姓名，`RoomName` 为教室名称，`DayOfTheWeek` 依然从 `0` 开始表示周一，`TimeOfTheDay` 使用逗号合并当天有课的节次。

### 生成 ICS 日历

该功能继承自原 [getMyCourses](https://github.com/whoisnian/getMyCourses) 项目的课表转日历思路，用于生成可导入 Apple Calendar、Google Calendar、Outlook 等日历软件的 `.ics` 内容。与原项目不同的是，WeCourseService 不把某个校区的作息时间写死在代码里，而是通过 `ClassTimeSlots` 按学校实际上课时间生成事件。

请求格式：

```json
{
	"Type": "ics",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```text
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Ares-Gao//WeCourseService//CN
CALSCALE:GREGORIAN
METHOD:PUBLISH
X-WR-CALNAME:微课表
X-WR-TIMEZONE:Asia/Shanghai
BEGIN:VEVENT
UID:14290-A080311-4-11--1-3-4@wecourse.service
DTSTAMP:20260604T120000Z
DTSTART;TZID=Asia/Shanghai:20200909T140000
DTEND;TZID=Asia/Shanghai:20200909T154000
SUMMARY:JavaScript程序设计
LOCATION:301\,计算机基础实训室(一)
DESCRIPTION:CourseID: 14290(A080311-4.11)\nRoomID: 1526
END:VEVENT
END:VCALENDAR
```

ICS 的每个事件会根据 `CalendarFirst`、`Weeks`、`DayOfTheWeek` 和 `ClassTimeSlots` 计算真实日期与起止时间。若学校作息时间和示例不同，请先修改 `ClassTimeSlots`，否则导入日历后的上课时间会不准确。

### 获取学籍照片

```json
{
	"Type": "photo",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回示例：

```text
data:image/jpg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/...NFaXMND/9k=
```

请遵守个人信息保护和肖像权相关法律法规。

### 获取成绩

本功能用于查询历史成绩。

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
	},
	{
		"CourseID": "A080011-6",
		"CourseName": "Java程序设计",
		"CourseTerm": "2018-2019 2",
		"CourseCredit": "6",
		"CourseGrade": "95",
		"GradePoint": "4.5"
	},
	{
		"CourseID": "A080310-6",
		"CourseName": "JavaWeb程序设计",
		"CourseTerm": "2019-2020 1",
		"CourseCredit": "6",
		"CourseGrade": "93",
		"GradePoint": "4.5"
	}
]
```

返回的子项全部为字符串类型。`CourseID` 表示课程 ID，`CourseName` 是课程名称，`CourseTerm` 代表学期，例如 `2019-2020 1` 表示 2019-2020 学年第一学期，`CourseCredit` 代表学分，`CourseGrade` 代表最终成绩，`GradePoint` 代表绩点。

## 许可证

本项目使用 MIT License，详见 [LICENSE](./LICENSE)。

项目已获得软件著作权，登记号：`2019SR0620279`。

## 致谢

- 感谢 OpenAI Codex 与 ChatGPT 对本次重写、整理和调试工作的协助。
- 感谢原 [getMyCourses](https://github.com/whoisnian/getMyCourses) 项目提供的基础思路。
- 感谢智慧陕理项目：[smartsnut.cn](https://smartsnut.cn/)。
