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
│   ├── internal/
│   ├── config.json
│   └── go.mod
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
	"DdddOcrUseCustomModel": false
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

Go 版本默认读取 `go/config.json`。Python 版本默认读取仓库根目录的 `config.json`，也可以通过 `WECOURSE_CONFIG` 指定配置文件路径。

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

当前 authserver 登录分支已接入 Python、C#、PHP、Java SDK。Go 版本已补充配置字段，后续会继续把分散在各接口中的登录流程收敛为统一 helper 后接入该模式。

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

通用返回格式：

```json
{
	"Type": "course",
	"Data": {}
}
```

`Type` 与请求中的 `Type` 保持一致，`Data` 为接口返回数据。

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

```json
{
	"Type": "login",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回：`登录成功` 或 `登录失败`。

### 获取当前教学周

```json
{
	"Type": "week"
}
```

返回：当前教学周数字。

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

```json
{
	"Type": "account",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

请遵守个人信息保护相关法律法规。无明确业务需要时，不要存储或缓存用户个人信息。

### 获取课程表

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

### 获取学籍照片

```json
{
	"Type": "photo",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

返回：`data:image/jpg;base64,...` 格式的图片数据。请遵守个人信息保护和肖像权相关法律法规。

### 获取成绩

```json
{
	"Type": "grade",
	"UserName": "201808830303",
	"PassWord": "7355608"
}
```

## 许可证

本项目使用 MIT License，详见 [LICENSE](./LICENSE)。

项目已获得软件著作权，登记号：`2019SR0620279`。

## 致谢

- 感谢 OpenAI Codex 与 ChatGPT 对本次重写、整理和调试工作的协助。
- 感谢原 [getMyCourses](https://github.com/whoisnian/getMyCourses) 项目提供的基础思路。
- 感谢智慧陕理项目：[smartsnut.cn](https://smartsnut.cn/)。
