# WeCourseService C# 版本

这是 WeCourseService 的 C# SDK 实现，面向 .NET 项目直接集成使用。当前提供树维教务系统核心查询能力，后续可继续扩展 HTTP API 或 WebSocket 服务包装。

## 环境要求

- .NET 8+

## 使用

```bash
cd csharp
dotnet build
```

示例：

```csharp
using WeCourseServiceSdk;

var config = WeCourseConfig.Load("../config.json");
var client = new SupwisdomClient(config);
var weekJson = client.GetWeekTime();
```

如果学校启用了普通图片验证码，实现 `ICaptchaSolver` 并传入客户端：

```csharp
var client = new SupwisdomClient(config, new MyCaptchaSolver());
```

### Authserver 登录

统一身份认证学校可在配置中增加：

```json
{
  "LoginType": "authserver",
  "AuthServerURL": "https://authserver.snut.edu.cn/authserver/login?service=http%3A%2F%2Fjwgl.snut.edu.cn%2Feams%2FssoLogin.action",
  "ServiceURL": "http://jwgl.snut.edu.cn/eams/ssoLogin.action"
}
```

验证码可接入 Tesseract、PaddleOCR 服务或自建 OCR 服务。

## 支持能力

- 登录验证
- 识别账号身份
- 获取教师课表
- 获取教师考试安排
- 查询教师考试批次
- 查询公共空闲教室
- 获取当前教学周
- 查询当前学期 ID 与课表参数
- 获取本学期课程
- 获取指定周课程
- 生成 ICS 日历
- 获取教师列表
- 获取学籍信息
- 获取学籍照片
- 获取成绩
