# WeCourseService Python 版本

这是 WeCourseService 的 Python 实现，协议和返回结构与 Go 版本保持一致，方便使用者按自己的技术栈选择。

## 环境要求

- Python 3.8+
- `requests`
- `websockets`

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

## 启动

```bash
python -m wecourse_service
```

服务会监听 `config.json` 中的 `SocketPort`。

## 支持接口

- `login`
- `week`
- `teacher`
- `account`
- `allcourse`
- `course`
- `weekcourse`
- `daycourse`
- `photo`
- `grade`
