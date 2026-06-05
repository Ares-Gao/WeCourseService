# WeCourseService PHP 版本

这是 WeCourseService 的 PHP SDK 实现，适合在传统 Web 项目、Laravel/Symfony 项目或轻量脚本中直接集成。

## 环境要求

- PHP 8.1+
- ext-curl
- ext-json

## 使用

使用 Composer：

```bash
cd php
composer install
```

```php
require __DIR__ . '/vendor/autoload.php';

$config = WeCourseConfig::load(__DIR__ . '/../config.json');
$client = new SupwisdomClient($config);
echo $client->getWeekTime();
echo $client->getCourse('username', 'password', 'authserver', $config->AuthServerURL);
echo $client->getIcs('username', 'password');
```

也可以不使用 Composer，直接 require `src/WeCourseConfig.php` 和 `src/SupwisdomClient.php`。

如果学校启用了普通图片验证码，可传入 callable：

```php
$client = new SupwisdomClient($config, function (string $imageBytes): string {
    $file = tempnam(sys_get_temp_dir(), 'captcha_') . '.jpg';
    file_put_contents($file, $imageBytes);
    $code = trim(shell_exec('tesseract ' . escapeshellarg($file) . ' stdout'));
    @unlink($file);
    return $code;
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

验证码可接入 Tesseract CLI、PaddleOCR 服务或自建 OCR 服务。

## 支持能力

- 登录验证
- 识别账号身份
- 获取教师课表
- 获取教师考试安排
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
