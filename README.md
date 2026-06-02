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

后续计划会继续整理 JavaScript/TypeScript、Java、C#、PHP、Rust 等主流语言版本。具体实现会按稳定性逐步加入仓库。

## 功能

- 验证教务系统账号登录
- 获取当前教学周
- 获取本学期课表、指定周课表、当天课表
- 获取教师列表
- 获取学籍信息
- 获取学籍照片
- 获取成绩
- 缓存课表结果，减少对教务系统的频繁访问
- 逐步提供多语言 SDK 与多种接入方式

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

## 配置

配置文件示例：

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

Go 版本默认读取 `go/config.json`。Python 版本默认读取仓库根目录的 `config.json`，也可以通过 `WECOURSE_CONFIG` 指定配置文件路径。

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
