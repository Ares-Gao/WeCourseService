# WeCourseService Python 版本

这是 WeCourseService 的 Python 实现，协议和返回结构与 Go 版本保持一致，方便使用者按自己的技术栈选择。

## 环境要求

- Python 3.8+
- `requests`
- `websockets`
- `pycryptodome`
- `ddddocr`

## 安装依赖

```bash
cd python
python -m venv .venv
.venv\Scripts\activate
pip install -r requirements.txt
```

Linux/macOS 激活虚拟环境：

```bash
source .venv/bin/activate
```

## 配置

Python 版本默认读取仓库根目录的 `config.json`。也可以通过环境变量指定配置文件：

```bash
set WECOURSE_CONFIG=..\config.json
```

Linux/macOS：

```bash
export WECOURSE_CONFIG=../config.json
```

### Authserver 登录

统一身份认证学校可在配置中启用：

```json
{
	"LoginType": "authserver",
	"AuthServerURL": "https://authserver.snut.edu.cn/authserver/login?service=http%3A%2F%2Fjwgl.snut.edu.cn%2Feams%2FssoLogin.action",
	"ServiceURL": "http://jwgl.snut.edu.cn/eams/ssoLogin.action",
	"AuthServerAutoCaptcha": true,
	"AuthServerCaptchaRetries": 3
}
```

其余字段仍需保留，例如 `MangerURL` 应填写教务系统根地址。

Python 版本会在 authserver 模式下自动调用 `checkNeedCaptcha.htl` 判断是否需要验证码。若需要普通图片验证码，会拉取 `getCaptcha.htl` 并使用 `ddddocr` 自动识别，失败时按 `AuthServerCaptchaRetries` 重试。

ICS 日历生成会读取 `CalendarTimezone`、`CalendarName` 和 `ClassTimeSlots`。请按学校真实作息配置 `ClassTimeSlots`，避免导入日历后的课程时间不准确。

`LoginType` 和 `AuthServerURL` 是服务配置项。业务请求不需要重复传递这些字段；同一条 WebSocket 连接先发送一次 `login`，后续请求只传 `Type` 和业务参数。

## 启动

```bash
python -m wecourse_service
```

服务会监听 `config.json` 中的 `SocketPort`。

## 支持接口

- `login`
- `identity`
- `teachercourse`
- `teacherexam`
- `teacherexambatch`
- `freeroom`
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
- `ics`
