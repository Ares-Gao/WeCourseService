# WeCourseService Java 版本

这是 WeCourseService 的 Java SDK 实现，适合 Spring、普通 JVM 服务或命令行工具集成。当前使用 JDK 标准库实现，不依赖第三方包。

## 环境要求

- JDK 17+

## 编译

使用 Maven：

```bash
cd java
mvn test
```

或直接使用 JDK：

```bash
cd java
javac -encoding UTF-8 -d out src/main/java/io/github/aresgao/wecourseservice/*.java
```

示例：

```java
var config = WeCourseConfig.load("../config.json");
var client = new SupwisdomClient(config);
System.out.println(client.getWeekTime());
System.out.println(client.getCourse("username", "password"));
System.out.println(client.getIcs("username", "password"));
```

如果学校启用了普通图片验证码，实现 `CaptchaSolver` 并传入客户端：

```java
var client = new SupwisdomClient(config, imageBytes -> {
    return "";
});
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

验证码可接入 Tess4J、PaddleOCR 服务或自建 OCR 服务。

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
