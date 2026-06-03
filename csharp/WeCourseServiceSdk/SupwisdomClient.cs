using System.Net;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.RegularExpressions;

namespace WeCourseServiceSdk;

public sealed class SupwisdomClient
{
    private const string UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0";
    private const string LoginMarker = "<a href=\"/eams/security/my.action\" target=\"_blank\" title=\"查看详情\" style=\"color:#ffffff\">";
    private const string AesChars = "ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678";
    private readonly WeCourseConfig _config;
    private readonly ICaptchaSolver? _captchaSolver;
    private readonly Dictionary<string, (DateTimeOffset CreatedAt, string Json, IReadOnlyList<Teacher> Teachers)> _courseCache = new();

    public SupwisdomClient(WeCourseConfig config, ICaptchaSolver? captchaSolver = null)
    {
        _config = config;
        _captchaSolver = captchaSolver;
    }

    public string Login(string username, string password)
    {
        try
        {
            using var session = CreateLoggedInClient(username, password);
            return JsonResponse("login", "登录成功");
        }
        catch
        {
            return JsonResponse("login", "登录失败");
        }
    }

    public string GetWeekTime()
    {
        var start = DateTime.Parse(_config.CalendarFirst + " 00:00:00");
        var week = (int)Math.Round((DateTime.Now - start).TotalDays / 7) + 1;
        return JsonResponse("week", week.ToString());
    }

    public string GetSemesters(string username, string password)
    {
        using var client = CreateLoggedInClient(username, password);
        var page = client.GetStringAsync(_config.BaseUrl + "eams/courseTableForStd.action").GetAwaiter().GetResult();
        var (ids, semesterId) = ExtractCourseTableParams(page);
        return JsonResponse("semester", new[] { new Semester(semesterId, ids, true) });
    }

    public string GetCourse(string username, string password)
    {
        if (_courseCache.TryGetValue(username, out var item) && DateTimeOffset.Now - item.CreatedAt < TimeSpan.FromHours(1))
        {
            return item.Json;
        }

        using var client = CreateLoggedInClient(username, password);
        var html = GetCourseTableHtml(client);
        var teachers = ParseTeachers(html);
        var courses = ParseCourses(html);
        var json = JsonResponse("allcourse", courses);
        _courseCache[username] = (DateTimeOffset.Now, json, teachers);
        return json;
    }

    public string GetTeacher(string username, string password)
    {
        using var client = CreateLoggedInClient(username, password);
        return JsonResponse("teacher", ParseTeachers(GetCourseTableHtml(client)));
    }

    public string GetWeekCourse(string username, string password, int week)
    {
        var courses = JsonSerializer.Deserialize<Response<List<Course>>>(GetCourse(username, password))?.Data ?? [];
        var teachers = _courseCache.TryGetValue(username, out var cache) ? cache.Teachers : [];
        var result = new List<WeekCourse>();

        foreach (var course in courses)
        {
            if (week >= course.Weeks.Length || course.Weeks[week] != '1')
            {
                continue;
            }

            foreach (var teacher in teachers)
            {
                if (!course.CourseID.Contains(teacher.CourseID))
                {
                    continue;
                }

                result.Add(new WeekCourse(
                    teacher.CourseName,
                    teacher.CourseTeacher,
                    course.RoomName,
                    course.CourseTimes.Count > 0 ? course.CourseTimes[0].DayOfTheWeek : 0,
                    string.Join(",", course.CourseTimes.Select(item => item.TimeOfTheDay + 1))));
            }
        }

        return JsonResponse("course", result);
    }

    public string GetAccount(string username, string password)
    {
        using var client = CreateLoggedInClient(username, password);
        var html = client.GetStringAsync(_config.BaseUrl + "eams/stdDetail.action").GetAwaiter().GetResult();
        var info = Regex.Matches(html, "(?i)<td>([^>]*)</td>").Select(item => item.Groups[1].Value).ToList();
        var student = new Student(info[0], info[1], info[2], info[11], info[12], info[4], $"{info[5]}({info[14]})", info[8], info[9], info[18]);
        return JsonResponse("account", student);
    }

    public string GetPhoto(string username, string password)
    {
        using var client = CreateLoggedInClient(username, password);
        var bytes = client.GetByteArrayAsync(_config.BaseUrl + $"eams/showSelfAvatar.action?user.name={username}").GetAwaiter().GetResult();
        return JsonResponse("photo", "data:image/jpg;base64," + Convert.ToBase64String(bytes));
    }

    public string GetGrade(string username, string password)
    {
        using var client = CreateLoggedInClient(username, password);
        var response = client.PostAsync(_config.BaseUrl + "eams/teach/grade/course/person!historyCourseGrade.action?projectType=MAJOR", new FormUrlEncodedContent([])).GetAwaiter().GetResult();
        response.EnsureSuccessStatusCode();
        var html = response.Content.ReadAsStringAsync().GetAwaiter().GetResult();
        var grades = new List<Grade>();

        foreach (Match row in Regex.Matches(html, "(?i)<tr>[\\s\\S]*?</tr>").Skip(2))
        {
            var cells = Regex.Matches(row.Value, "(?i)<td.*>([^>]*)</td>").Select(item => item.Groups[1].Value).ToList();
            if (cells.Count < 6)
            {
                continue;
            }

            var sup = Regex.Match(row.Value, "(?i)<sup.*>([^>]*)</sup>");
            grades.Add(new Grade(
                cells[1].Trim('\n'),
                sup.Success ? sup.Groups[1].Value : cells[3].Trim('\t', '\r', '\n'),
                cells[0].Trim('\n'),
                cells[4].Trim('\n'),
                cells[^2].Trim('\t', '\n'),
                cells[^1].Trim('\t', '\n')));
        }

        return JsonResponse("grade", grades);
    }

    private HttpClient CreateLoggedInClient(string username, string password)
    {
        if (string.Equals(_config.LoginType, "authserver", StringComparison.OrdinalIgnoreCase))
        {
            return CreateAuthServerLoggedInClient(username, password);
        }

        var handler = new HttpClientHandler { CookieContainer = new CookieContainer() };
        var client = new HttpClient(handler);
        client.DefaultRequestHeaders.UserAgent.ParseAdd(UserAgent);

        var loginHtml = client.GetStringAsync(_config.BaseUrl + "eams/login.action").GetAwaiter().GetResult();
        var salt = ExtractPasswordSalt(loginHtml);
        var hashedPassword = Sha1(salt + password);

        Thread.Sleep(1000);
        var response = client.PostAsync(
            _config.BaseUrl + "eams/login.action",
            new FormUrlEncodedContent(new Dictionary<string, string>
            {
                ["username"] = username,
                ["password"] = hashedPassword,
                ["session_locale"] = "zh_CN",
            })).GetAwaiter().GetResult();
        response.EnsureSuccessStatusCode();

        var content = response.Content.ReadAsStringAsync().GetAwaiter().GetResult();
        if (!content.Contains(LoginMarker))
        {
            throw new InvalidOperationException("Login failed.");
        }

        return client;
    }

    private HttpClient CreateAuthServerLoggedInClient(string username, string password)
    {
        if (string.IsNullOrWhiteSpace(_config.AuthServerURL))
        {
            throw new InvalidOperationException("AuthServerURL is required for authserver login.");
        }

        var handler = new HttpClientHandler { CookieContainer = new CookieContainer() };
        var client = new HttpClient(handler);
        client.DefaultRequestHeaders.UserAgent.ParseAdd(UserAgent);

        var loginHtml = client.GetStringAsync(_config.AuthServerURL).GetAwaiter().GetResult();
        var salt = InputValue(loginHtml, elementId: "pwdEncryptSalt");
        var execution = InputValue(loginHtml, name: "execution");
        if (string.IsNullOrWhiteSpace(salt) || string.IsNullOrWhiteSpace(execution))
        {
            throw new InvalidOperationException("Authserver login page is missing pwdEncryptSalt or execution.");
        }

        var captcha = "";
        if (NeedAuthServerCaptcha(loginHtml))
        {
            if (_captchaSolver is null)
            {
                throw new InvalidOperationException("Authserver captcha is required. Provide an ICaptchaSolver implementation.");
            }
            var captchaBytes = client.GetByteArrayAsync(AuthServerBase(_config.AuthServerURL) + "/getCaptcha.htl?" + DateTimeOffset.Now.ToUnixTimeMilliseconds()).GetAwaiter().GetResult();
            captcha = _captchaSolver.Solve(captchaBytes);
        }

        var response = client.PostAsync(_config.AuthServerURL, new FormUrlEncodedContent(new Dictionary<string, string>
        {
            ["username"] = username,
            ["password"] = AuthServerEncryptPassword(password, salt),
            ["captcha"] = captcha,
            ["_eventId"] = InputValue(loginHtml, name: "_eventId") is { Length: > 0 } eventId ? eventId : "submit",
            ["cllt"] = "userNameLogin",
            ["dllt"] = InputValue(loginHtml, name: "dllt") is { Length: > 0 } dllt ? dllt : "generalLogin",
            ["lt"] = InputValue(loginHtml, name: "lt"),
            ["execution"] = execution,
            ["rmShown"] = "1",
        })).GetAwaiter().GetResult();
        var content = response.Content.ReadAsStringAsync().GetAwaiter().GetResult();
        var finalPath = response.RequestMessage?.RequestUri?.AbsolutePath ?? "";
        if (finalPath.Contains("/authserver/login") && !response.IsSuccessStatusCode)
        {
            response.EnsureSuccessStatusCode();
        }
        if (finalPath.Contains("/authserver/login") || content.Contains("认证失败"))
        {
            throw new InvalidOperationException("Authserver login failed.");
        }

        return client;
    }

    private string GetCourseTableHtml(HttpClient client)
    {
        Thread.Sleep(1000);
        var page = client.GetStringAsync(_config.BaseUrl + "eams/courseTableForStd.action").GetAwaiter().GetResult();
        var (ids, semesterId) = ExtractCourseTableParams(page);
        var response = client.PostAsync(
            _config.BaseUrl + "eams/courseTableForStd!courseTable.action",
            new FormUrlEncodedContent(new Dictionary<string, string>
            {
                ["ignoreHead"] = "1",
                ["showPrintAndExport"] = "1",
                ["setting.kind"] = "std",
                ["startWeek"] = "",
                ["semester.id"] = semesterId,
                ["ids"] = ids,
            })).GetAwaiter().GetResult();
        response.EnsureSuccessStatusCode();
        return response.Content.ReadAsStringAsync().GetAwaiter().GetResult();
    }

    private static IReadOnlyList<Teacher> ParseTeachers(string html)
    {
        var result = new List<Teacher>();
        var rows = Regex.Matches(html, @"(?i)<td>(\d)</td>\s*<td>([:alpha:].+)</td>\s*<td>(.+)</td>\s*<td>((\d)|(\d\.\d))</td>\s*<td>\s*<a href=.*\s.*\s.*\s.*>.*</a>\s*</td>\s*<td>(.*)</td>");

        foreach (Match row in rows)
        {
            var cells = Regex.Matches(row.Value, "(?i)<td>([^>]*)</td>").Select(item => item.Groups[1].Value).ToList();
            var links = Regex.Matches(row.Value, "(?i)>([^>]*)</a>").Select(item => item.Groups[1].Value).ToList();
            if (cells.Count >= 5 && links.Count > 0)
            {
                result.Add(new Teacher(links[0], cells[2], cells[3], cells[4]));
            }
        }

        return result;
    }

    private static IReadOnlyList<Course> ParseCourses(string html)
    {
        var result = new List<Course>();
        var rows = Regex.Matches(html, "TaskActivity\\(actTeacherId.join\\(','\\),actTeacherName.join\\(','\\),\"(.*)\",\"(.*)\\(.*\\)\",\"(.*)\",\"(.*)\",\"(.*)\",null,null,assistantName,\"\",\"\"\\);((?:\\s*index =\\d+\\*unitCount\\+\\d+;\\s*.*\\s)+)");

        foreach (Match row in rows)
        {
            var times = new List<CourseTime>();
            foreach (var indexText in row.Groups[6].Value.Split("table0.activities[index][table0.activities[index].length]=activity;"))
            {
                var index = Regex.Match(indexText, @"\s*index =(\d+)\*unitCount\+(\d+);\s*");
                if (index.Success)
                {
                    times.Add(new CourseTime(int.Parse(index.Groups[1].Value), int.Parse(index.Groups[2].Value)));
                }
            }

            result.Add(new Course(row.Groups[1].Value, CleanJsText(row.Groups[2].Value), row.Groups[3].Value, row.Groups[4].Value, row.Groups[5].Value, times));
        }

        return result;
    }

    private static (string Ids, string SemesterId) ExtractCourseTableParams(string html)
    {
        var ids = Regex.Match(html, "bg\\.form\\.addInput\\(form,\\s*\"ids\",\\s*\"([^\"]+)\"");
        if (!ids.Success)
        {
            throw new InvalidOperationException("Course table ids not found.");
        }

        var semesterPatterns = new[]
        {
            "name=[\"']semester\\.id[\"'][^>]*value=[\"']([^\"']+)[\"']",
            "semesterCalendar\\(\\{[^}]*value:\\s*\"([^\"]+)\"",
            "semesterCalendar\\(\\{[^}]*value:\\s*'([^']+)'",
            "bg\\.form\\.addInput\\(form,\\s*\"semester\\.id\",\\s*\"([^\"]+)\"",
        };
        foreach (var pattern in semesterPatterns)
        {
            var semester = Regex.Match(html, pattern, RegexOptions.IgnoreCase);
            if (semester.Success && !string.IsNullOrWhiteSpace(semester.Groups[1].Value))
            {
                return (ids.Groups[1].Value, semester.Groups[1].Value);
            }
        }

        throw new InvalidOperationException("semester.id not found.");
    }

    private static string CleanJsText(string text)
    {
        return text.Replace("\"+periodInfo+\"", "").Replace("\\\"", "\"");
    }

    private static string ExtractPasswordSalt(string html)
    {
        const string marker = "CryptoJS.SHA1(";
        var index = html.IndexOf(marker, StringComparison.Ordinal);
        if (index < 0)
        {
            throw new InvalidOperationException("Password salt not found.");
        }

        return html.Substring(index + 15, 37);
    }

    private static string Sha1(string value)
    {
        var hash = SHA1.HashData(Encoding.UTF8.GetBytes(value));
        return Convert.ToHexString(hash).ToLowerInvariant();
    }

    private static string AuthServerEncryptPassword(string password, string salt)
    {
        using var aes = Aes.Create();
        aes.Key = Encoding.UTF8.GetBytes(salt.Trim());
        aes.IV = Encoding.UTF8.GetBytes(RandomString(16));
        aes.Mode = CipherMode.CBC;
        aes.Padding = PaddingMode.PKCS7;
        var payload = Encoding.UTF8.GetBytes(RandomString(64) + password);
        return Convert.ToBase64String(aes.CreateEncryptor().TransformFinalBlock(payload, 0, payload.Length));
    }

    private static bool NeedAuthServerCaptcha(string html)
    {
        if (Regex.IsMatch(html, "var\\s+_badCredentialsCount\\s*=\\s*\"0\""))
        {
            return true;
        }
        return html.Contains("getCaptcha.htl") && html.Contains("captchaDiv");
    }

    private static string AuthServerBase(string loginUrl)
    {
        if (!Uri.TryCreate(loginUrl, UriKind.Absolute, out var uri))
        {
            return "";
        }
        var path = uri.AbsolutePath;
        var index = path.IndexOf("/login", StringComparison.Ordinal);
        var contextPath = index >= 0 ? path[..index] : "/authserver";
        return uri.GetLeftPart(UriPartial.Authority) + contextPath;
    }

    private static string RandomString(int length)
    {
        Span<byte> bytes = stackalloc byte[length];
        RandomNumberGenerator.Fill(bytes);
        var builder = new StringBuilder(length);
        foreach (var item in bytes)
        {
            builder.Append(AesChars[item % AesChars.Length]);
        }

        return builder.ToString();
    }

    private static string InputValue(string html, string name = "", string elementId = "")
    {
        var pattern = !string.IsNullOrEmpty(elementId)
            ? $"<input[^>]*id=[\\\"']{Regex.Escape(elementId)}[\\\"'][^>]*>"
            : $"<input[^>]*name=[\\\"']{Regex.Escape(name)}[\\\"'][^>]*>";
        var input = Regex.Match(html, pattern, RegexOptions.IgnoreCase).Value;
        return Regex.Match(input, "value=[\\\"']([^\\\"']*)", RegexOptions.IgnoreCase).Groups[1].Value;
    }

    private static string JsonResponse<T>(string type, T data)
    {
        return JsonSerializer.Serialize(new Response<T>(type, data), new JsonSerializerOptions { WriteIndented = true });
    }

    private sealed record Response<T>(string Type, T Data);
}
