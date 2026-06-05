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
    private static readonly IReadOnlyList<ClassTimeSlot> DefaultClassTimeSlots =
    [
        new("08:00", "08:45"),
        new("08:55", "09:40"),
        new("10:00", "10:45"),
        new("10:55", "11:40"),
        new("14:00", "14:45"),
        new("14:55", "15:40"),
        new("16:00", "16:45"),
        new("16:55", "17:40"),
        new("19:00", "19:45"),
        new("19:55", "20:40"),
        new("20:50", "21:35"),
        new("21:45", "22:30"),
    ];

    public SupwisdomClient(WeCourseConfig config, ICaptchaSolver? captchaSolver = null)
    {
        _config = config;
        _captchaSolver = captchaSolver;
    }

    public string Login(string username, string password, string loginType = "", string authServerUrl = "")
    {
        try
        {
            using var session = CreateLoggedInClient(username, password, loginType, authServerUrl);
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

    public string GetIdentity(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var html = client.GetStringAsync(_config.BaseUrl + "eams/homeExt.action").GetAwaiter().GetResult();
        return JsonResponse("identity", ParseHomeExtIdentity(html));
    }

    public string GetTeacherCourse(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        return JsonResponse("teachercourse", ParseCourses(GetTeacherCourseTableHtml(client)));
    }

    public string GetTeacherExam(string username, string password, string loginType = "", string authServerUrl = "", string examBatchId = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var page = client.GetStringAsync(_config.BaseUrl + "eams/teacherExamTable.action").GetAwaiter().GetResult();
        if (string.IsNullOrWhiteSpace(examBatchId))
        {
            examBatchId = ParseTeacherExamBatches(page).FirstOrDefault(batch => batch.Selected)?.ExamBatchID ?? "";
        }
        if (string.IsNullOrWhiteSpace(examBatchId))
        {
            return JsonResponse("teacherexam", Array.Empty<TeacherExam>());
        }
        var html = client.GetStringAsync(_config.BaseUrl + "eams/teacherExamTable!examAtivities.action?examBatch.id=" + Uri.EscapeDataString(examBatchId)).GetAwaiter().GetResult();
        return JsonResponse("teacherexam", ParseTeacherExams(html));
    }

    public string GetTeacherExamBatches(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var page = client.GetStringAsync(_config.BaseUrl + "eams/teacherExamTable.action").GetAwaiter().GetResult();
        return JsonResponse("teacherexambatch", ParseTeacherExamBatches(page));
    }

    public string GetFreeRoom(
        string dateBegin,
        string dateEnd = "",
        string timeBegin = "1",
        string timeEnd = "",
        string roomApplyTimeType = "0",
        string classroomType = "",
        string campusId = "",
        string buildingId = "",
        string seats = "",
        string classroomName = "")
    {
        if (string.IsNullOrWhiteSpace(dateBegin))
        {
            dateBegin = DateTime.Now.ToString("yyyy-MM-dd");
        }
        if (string.IsNullOrWhiteSpace(dateEnd))
        {
            dateEnd = dateBegin;
        }
        if (string.IsNullOrWhiteSpace(timeEnd))
        {
            timeEnd = timeBegin;
        }
        using var client = new HttpClient();
        var response = client.PostAsync(_config.BaseUrl + "eams/publicFree!search.action", new FormUrlEncodedContent(new Dictionary<string, string>
        {
            ["classroom.type.id"] = classroomType,
            ["classroom.campus.id"] = campusId,
            ["classroom.building.id"] = buildingId,
            ["seats"] = seats,
            ["classroom.name"] = classroomName,
            ["cycleTime.cycleCount"] = "1",
            ["cycleTime.cycleType"] = "1",
            ["cycleTime.dateBegin"] = dateBegin,
            ["cycleTime.dateEnd"] = dateEnd,
            ["roomApplyTimeType"] = roomApplyTimeType,
            ["timeBegin"] = timeBegin,
            ["timeEnd"] = timeEnd,
        })).GetAwaiter().GetResult();
        response.EnsureSuccessStatusCode();
        return JsonResponse("freeroom", ParseFreeRooms(response.Content.ReadAsStringAsync().GetAwaiter().GetResult()));
    }

    public string GetSemesters(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var page = client.GetStringAsync(_config.BaseUrl + "eams/courseTableForStd.action").GetAwaiter().GetResult();
        var (ids, semesterId) = ExtractCourseTableParams(page);
        return JsonResponse("semester", new[] { new Semester(semesterId, ids, true) });
    }

    public string GetCourse(string username, string password, string loginType = "", string authServerUrl = "")
    {
        var cacheKey = $"{(string.IsNullOrWhiteSpace(loginType) ? _config.LoginType : loginType)}:{authServerUrl}:{username}";
        if (_courseCache.TryGetValue(cacheKey, out var item) && DateTimeOffset.Now - item.CreatedAt < TimeSpan.FromHours(1))
        {
            return item.Json;
        }

        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var html = GetCourseTableHtml(client);
        var teachers = ParseTeachers(html);
        var courses = ParseCourses(html);
        var json = JsonResponse("allcourse", courses);
        _courseCache[cacheKey] = (DateTimeOffset.Now, json, teachers);
        return json;
    }

    public string GetTeacher(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        return JsonResponse("teacher", ParseTeachers(GetCourseTableHtml(client)));
    }

    public string GetWeekCourse(string username, string password, int week, string loginType = "", string authServerUrl = "")
    {
        var courses = JsonSerializer.Deserialize<Response<List<Course>>>(GetCourse(username, password, loginType, authServerUrl))?.Data ?? [];
        var cacheKey = $"{(string.IsNullOrWhiteSpace(loginType) ? _config.LoginType : loginType)}:{authServerUrl}:{username}";
        var teachers = _courseCache.TryGetValue(cacheKey, out var cache) ? cache.Teachers : [];
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

    public string GetIcs(string username, string password, string loginType = "", string authServerUrl = "")
    {
        var courses = JsonSerializer.Deserialize<Response<List<Course>>>(GetCourse(username, password, loginType, authServerUrl))?.Data ?? [];
        return JsonResponse("ics", GenerateIcs(courses));
    }

    public string GenerateIcs(IReadOnlyList<Course> courses)
    {
        var slots = _config.ClassTimeSlots is { Count: > 0 } ? _config.ClassTimeSlots : DefaultClassTimeSlots;
        var timezone = string.IsNullOrWhiteSpace(_config.CalendarTimezone) ? "Asia/Shanghai" : _config.CalendarTimezone;
        var calendarName = string.IsNullOrWhiteSpace(_config.CalendarName) ? _config.SchoolName + "课表" : _config.CalendarName;
        var firstMonday = DateTime.Parse(_config.CalendarFirst);
        var now = DateTime.UtcNow.ToString("yyyyMMdd'T'HHmmss'Z'");
        var builder = new StringBuilder();
        builder.Append("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Ares-Gao//WeCourseService//CN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n");
        builder.Append(FoldIcsLine("X-WR-CALNAME:" + EscapeIcsText(calendarName)));
        builder.Append(FoldIcsLine("X-WR-TIMEZONE:" + timezone));

        foreach (var course in courses)
        {
            foreach (var dayGroup in course.CourseTimes.GroupBy(item => item.DayOfTheWeek))
            {
                var times = dayGroup.Select(item => item.TimeOfTheDay).Order().ToList();
                var startSlot = times[0];
                var endSlot = times[^1];
                if (startSlot >= slots.Count || endSlot >= slots.Count)
                {
                    continue;
                }
                for (var weekIndex = 0; weekIndex < course.Weeks.Length; weekIndex++)
                {
                    if (weekIndex == 0 || course.Weeks[weekIndex] != '1')
                    {
                        continue;
                    }
                    var date = firstMonday.AddDays((weekIndex - 1) * 7 + dayGroup.Key);
                    var startAt = DateTime.Parse(date.ToString("yyyy-MM-dd") + " " + slots[startSlot].Start);
                    var endAt = DateTime.Parse(date.ToString("yyyy-MM-dd") + " " + slots[endSlot].End);
                    var uid = Regex.Replace($"{course.CourseID}-{weekIndex}-{dayGroup.Key}-{startSlot}", "[^0-9A-Za-z_-]", "-") + "@wecourse.service";

                    builder.Append("BEGIN:VEVENT\r\n");
                    builder.Append(FoldIcsLine("UID:" + uid));
                    builder.Append("DTSTAMP:" + now + "\r\n");
                    builder.Append(FoldIcsLine("DTSTART;TZID=" + timezone + ":" + startAt.ToString("yyyyMMdd'T'HHmmss")));
                    builder.Append(FoldIcsLine("DTEND;TZID=" + timezone + ":" + endAt.ToString("yyyyMMdd'T'HHmmss")));
                    builder.Append(FoldIcsLine("SUMMARY:" + EscapeIcsText(course.CourseName)));
                    builder.Append(FoldIcsLine("LOCATION:" + EscapeIcsText(course.RoomName)));
                    builder.Append(FoldIcsLine("DESCRIPTION:" + EscapeIcsText($"CourseID: {course.CourseID}\nRoomID: {course.RoomID}")));
                    builder.Append("END:VEVENT\r\n");
                }
            }
        }
        builder.Append("END:VCALENDAR\r\n");
        return builder.ToString();
    }

    public string GetAccount(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var html = client.GetStringAsync(_config.BaseUrl + "eams/stdDetail.action").GetAwaiter().GetResult();
        var info = Regex.Matches(html, "(?i)<td>([^>]*)</td>").Select(item => item.Groups[1].Value).ToList();
        var student = new Student(info[0], info[1], info[2], info[11], info[12], info[4], $"{info[5]}({info[14]})", info[8], info[9], info[18]);
        return JsonResponse("account", student);
    }

    public string GetPhoto(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
        var bytes = client.GetByteArrayAsync(_config.BaseUrl + $"eams/showSelfAvatar.action?user.name={username}").GetAwaiter().GetResult();
        return JsonResponse("photo", "data:image/jpg;base64," + Convert.ToBase64String(bytes));
    }

    public string GetGrade(string username, string password, string loginType = "", string authServerUrl = "")
    {
        using var client = CreateLoggedInClient(username, password, loginType, authServerUrl);
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

    private HttpClient CreateLoggedInClient(string username, string password, string loginType = "", string authServerUrl = "")
    {
        var resolvedLoginType = string.IsNullOrWhiteSpace(loginType) ? _config.LoginType : loginType;
        if (string.Equals(resolvedLoginType, "authserver", StringComparison.OrdinalIgnoreCase))
        {
            return CreateAuthServerLoggedInClient(username, password, authServerUrl);
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

    private HttpClient CreateAuthServerLoggedInClient(string username, string password, string authServerUrl = "")
    {
        var loginUrl = string.IsNullOrWhiteSpace(authServerUrl) ? _config.AuthServerURL : authServerUrl;
        if (string.IsNullOrWhiteSpace(loginUrl))
        {
            throw new InvalidOperationException("AuthServerURL is required for authserver login.");
        }

        var handler = new HttpClientHandler { CookieContainer = new CookieContainer() };
        var client = new HttpClient(handler);
        client.DefaultRequestHeaders.UserAgent.ParseAdd(UserAgent);

        var loginHtml = client.GetStringAsync(loginUrl).GetAwaiter().GetResult();
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
            var captchaBytes = client.GetByteArrayAsync(AuthServerBase(loginUrl) + "/getCaptcha.htl?" + DateTimeOffset.Now.ToUnixTimeMilliseconds()).GetAwaiter().GetResult();
            captcha = _captchaSolver.Solve(captchaBytes);
        }

        var response = client.PostAsync(loginUrl, new FormUrlEncodedContent(new Dictionary<string, string>
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

    private string GetTeacherCourseTableHtml(HttpClient client)
    {
        var page = client.GetStringAsync(_config.BaseUrl + "eams/courseTableForTeacher.action").GetAwaiter().GetResult();
        var ids = Regex.Match(page, "name=[\"']ids[\"'][^>]*value=[\"']([^\"']+)[\"']");
        var semester = Regex.Match(page, "semesterCalendar\\(\\{[^}]*value:[\"']([^\"']+)[\"']");
        if (!ids.Success || !semester.Success)
        {
            throw new InvalidOperationException("Teacher course table params not found.");
        }
        var response = client.PostAsync(_config.BaseUrl + "eams/courseTableForTeacher!courseTable.action", new FormUrlEncodedContent(new Dictionary<string, string>
        {
            ["ignoreHead"] = "1",
            ["setting.forSemester"] = "1",
            ["ids"] = ids.Groups[1].Value,
            ["setting.kind"] = "teacher",
            ["semester.id"] = semester.Groups[1].Value,
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
        var rows = Regex.Matches(html, "TaskActivity\\(actTeacherId(?:\\.toString\\(\\)|.join\\(','\\)),[^,]*,\"(.*)\",\"(.*)\\(.*\\)\",\"(.*)\",\"(.*)\",\"(.*)\",null,null,assistantName,\"\",\"\"\\);((?:\\s*index =\\d+\\*unitCount\\+\\d+;\\s*.*\\s)+)");

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

    private static Identity ParseHomeExtIdentity(string html)
    {
        var categoryMatch = Regex.Match(html, @"<input[^>]+name=[""']security\.userCategoryId[""'][^>]*value=[""']([^""']+)[""']", RegexOptions.IgnoreCase | RegexOptions.Singleline);
        if (categoryMatch.Success)
        {
            var category = categoryMatch.Groups[1].Value.Trim();
            return category switch
            {
                "1" => new Identity("student", "学生", category),
                "2" => new Identity("teacher", "教师", category),
                _ => new Identity("unknown", "未知", category),
            };
        }

        if (html.Contains("courseTableForStd.action") || html.Contains("stdDetail.action") || html.Contains("学生"))
        {
            return new Identity("student", "学生", "");
        }
        if (html.Contains("courseTableForTeacher.action") || html.Contains("teacherExamTable.action") || html.Contains("教师"))
        {
            return new Identity("teacher", "教师", "");
        }
        return new Identity("unknown", "未知", "");
    }

    private static IReadOnlyList<TeacherExam> ParseTeacherExams(string html)
    {
        var result = new List<TeacherExam>();
        var sections = Regex.Split(html, "(?=<div id=\"toolbar[^\"]*\")", RegexOptions.IgnoreCase);
        foreach (var section in sections)
        {
            var title = CleanHtmlCell(Regex.Match(section, "bg\\.ui\\.toolbar\\(\"[^\"]+\",'([^']*)'").Groups[1].Value);
            foreach (var cells in TableRows(section))
            {
                if (cells.Count < 7 || string.IsNullOrWhiteSpace(cells[0]))
                {
                    continue;
                }
                if (cells.Count >= 8)
                {
                    result.Add(new TeacherExam(title, cells[0], cells[1], cells[2], cells[3], cells[5], cells[4], cells[6], cells[7]));
                }
                else
                {
                    result.Add(new TeacherExam(title, cells[0], cells[1], cells[2], cells[3], cells[4], "", cells[5], cells[6]));
                }
            }
        }
        return result;
    }

    private static IReadOnlyList<TeacherExamBatch> ParseTeacherExamBatches(string html)
    {
        var result = new List<TeacherExamBatch>();
        foreach (Match option in Regex.Matches(html, @"(?is)<option\s+value=[""']([^""']+)[""']([^>]*)>(.*?)</option>"))
        {
            result.Add(new TeacherExamBatch(
                option.Groups[1].Value,
                CleanHtmlCell(option.Groups[3].Value),
                option.Groups[2].Value.Contains("selected", StringComparison.OrdinalIgnoreCase)));
        }
        return result;
    }

    private static IReadOnlyList<FreeRoom> ParseFreeRooms(string html)
    {
        return TableRows(html)
            .Where(cells => cells.Count >= 6 && !string.IsNullOrWhiteSpace(cells[0]))
            .Select(cells => new FreeRoom(cells[0], cells[1], cells[2], cells[3], cells[4], cells[5]))
            .ToList();
    }

    private static IReadOnlyList<List<string>> TableRows(string html)
    {
        var rows = new List<List<string>>();
        foreach (Match row in Regex.Matches(html, "(?is)<tr[^>]*>(.*?)</tr>"))
        {
            var cells = Regex.Matches(row.Groups[1].Value, "(?is)<td[^>]*>(.*?)</td>")
                .Select(cell => CleanHtmlCell(cell.Groups[1].Value))
                .ToList();
            if (cells.Count > 0)
            {
                rows.Add(cells);
            }
        }
        return rows;
    }

    private static string CleanHtmlCell(string value)
    {
        value = Regex.Replace(value, "(?is)<[^>]+>", "");
        value = WebUtility.HtmlDecode(value).Replace("\u00a0", " ");
        return string.Join(" ", value.Split((char[]?)null, StringSplitOptions.RemoveEmptyEntries));
    }

    private static string CleanJsText(string text)
    {
        return text.Replace("\"+periodInfo+\"", "").Replace("\\\"", "\"");
    }

    private static string EscapeIcsText(string value)
    {
        return value.Replace("\\", "\\\\").Replace("\n", "\\n").Replace("\r", "").Replace(";", "\\;").Replace(",", "\\,");
    }

    private static string FoldIcsLine(string line)
    {
        if (line.Length <= 75)
        {
            return line + "\r\n";
        }
        var builder = new StringBuilder();
        for (var index = 0; index < line.Length; index += 75)
        {
            if (index > 0)
            {
                builder.Append("\r\n ");
            }
            builder.Append(line.Substring(index, Math.Min(75, line.Length - index)));
        }
        builder.Append("\r\n");
        return builder.ToString();
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
