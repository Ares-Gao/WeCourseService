# WeCourseService Go 版本

这是 WeCourseService 的 Go 实现，适合需要单文件部署、低运行时依赖或已有 Go 技术栈的场景。

## 环境要求

- Go 1.22+

## 配置

Go 版本默认读取当前工作目录下的 `config.json`。本目录已保留一份配置文件，部署前请按学校实际信息修改。

## 启动

```bash
cd go
go run ./cmd/wecourse-service
```

## 支持接口

- `login`
- `week`
- `semester`
- `teacher`
- `account`
- `allcourse`
- `course`
- `weekcourse`
- `daycourse`
- `photo`
- `grade`

构建：

```bash
cd go
go build -o bin/wecourse-service ./cmd/wecourse-service
```

## 结构

```text
.
├── cmd/
│   └── wecourse-service/     # 服务入口
├── internal/
│   └── service/              # WebSocket、配置读取和教务系统业务逻辑
├── config.json
└── go.mod
```
