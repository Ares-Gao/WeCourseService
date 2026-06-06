package io.github.aresgao.wecourseservice;

import java.io.IOException;
import java.net.CookieManager;
import java.net.CookiePolicy;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.SecureRandom;
import java.time.Duration;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;
import javax.crypto.Cipher;
import javax.crypto.spec.IvParameterSpec;
import javax.crypto.spec.SecretKeySpec;

public final class SupwisdomClient {
    private static final String USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0";
    private static final String LOGIN_MARKER = "<a href=\"/eams/security/my.action\" target=\"_blank\" title=\"查看详情\" style=\"color:#ffffff\">";
    private static final String AES_CHARS = "ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678";
    private static final SecureRandom RANDOM = new SecureRandom();

    private final WeCourseConfig config;
    private final CaptchaSolver captchaSolver;
    private final Map<String, CacheItem> courseCache = new HashMap<>();
    private static final List<ClassTimeSlot> DEFAULT_CLASS_TIME_SLOTS = List.of(
            new ClassTimeSlot("08:00", "08:45"),
            new ClassTimeSlot("08:55", "09:40"),
            new ClassTimeSlot("10:00", "10:45"),
            new ClassTimeSlot("10:55", "11:40"),
            new ClassTimeSlot("14:00", "14:45"),
            new ClassTimeSlot("14:55", "15:40"),
            new ClassTimeSlot("16:00", "16:45"),
            new ClassTimeSlot("16:55", "17:40"),
            new ClassTimeSlot("19:00", "19:45"),
            new ClassTimeSlot("19:55", "20:40"),
            new ClassTimeSlot("20:50", "21:35"),
            new ClassTimeSlot("21:45", "22:30"));

    public SupwisdomClient(WeCourseConfig config) {
        this(config, null);
    }

    public SupwisdomClient(WeCourseConfig config, CaptchaSolver captchaSolver) {
        this.config = config;
        this.captchaSolver = captchaSolver;
    }

    public String login(String username, String password) {
        return login(username, password, "", "");
    }

    public String login(String username, String password, String loginType, String authServerUrl) {
        try {
            createLoggedInClient(username, password, loginType, authServerUrl);
            return jsonResponse("login", "\"登录成功\"");
        } catch (Exception ex) {
            return jsonResponse("login", "\"登录失败\"");
        }
    }

    public String getWeekTime() {
        var start = LocalDateTime.parse(config.CalendarFirst() + " 00:00:00", DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"));
        var week = Math.round(Duration.between(start, LocalDateTime.now()).toDays() / 7.0) + 1;
        return jsonResponse("week", quote(Long.toString(week)));
    }

    public String getIdentity(String username, String password) throws Exception {
        return getIdentity(username, password, "", "");
    }

    public String getIdentity(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var html = get(client, config.baseUrl() + "eams/homeExt.action");
        var identity = parseHomeExtIdentity(html);
        return jsonResponse("identity", "{\"Role\":" + quote(identity.Role()) + ",\"RoleName\":" + quote(identity.RoleName()) + ",\"UserCategoryID\":" + quote(identity.UserCategoryID()) + "}");
    }

    public String getTeacherCourse(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        return jsonResponse("teachercourse", coursesJson(parseCourses(teacherCourseTableHtml(client))));
    }

    public String getTeacherExam(String username, String password, String loginType, String authServerUrl, String examBatchId) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var page = get(client, config.baseUrl() + "eams/teacherExamTable.action");
        if (examBatchId == null || examBatchId.isBlank()) {
            for (var batch : parseTeacherExamBatches(page)) {
                if (batch.Selected()) {
                    examBatchId = batch.ExamBatchID();
                    break;
                }
            }
        }
        if (examBatchId == null || examBatchId.isBlank()) {
            return jsonResponse("teacherexam", "[]");
        }
        var html = get(client, config.baseUrl() + "eams/teacherExamTable!examAtivities.action?examBatch.id=" + URLEncoder.encode(examBatchId, StandardCharsets.UTF_8));
        return jsonResponse("teacherexam", teacherExamsJson(parseTeacherExams(html)));
    }

    public String getTeacherExamBatches(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var page = get(client, config.baseUrl() + "eams/teacherExamTable.action");
        return jsonResponse("teacherexambatch", teacherExamBatchesJson(parseTeacherExamBatches(page)));
    }

    public String getFreeRoom(String dateBegin, String dateEnd, String timeBegin, String timeEnd) throws Exception {
        if (dateBegin == null || dateBegin.isBlank()) {
            dateBegin = LocalDate.now().toString();
        }
        if (dateEnd == null || dateEnd.isBlank()) {
            dateEnd = dateBegin;
        }
        if (timeBegin == null || timeBegin.isBlank()) {
            timeBegin = "1";
        }
        if (timeEnd == null || timeEnd.isBlank()) {
            timeEnd = timeBegin;
        }
        var client = HttpClient.newHttpClient();
        var html = post(client, config.baseUrl() + "eams/publicFree!search.action", form(Map.ofEntries(
                Map.entry("classroom.type.id", ""),
                Map.entry("classroom.campus.id", ""),
                Map.entry("classroom.building.id", ""),
                Map.entry("seats", ""),
                Map.entry("classroom.name", ""),
                Map.entry("cycleTime.cycleCount", "1"),
                Map.entry("cycleTime.cycleType", "1"),
                Map.entry("cycleTime.dateBegin", dateBegin),
                Map.entry("cycleTime.dateEnd", dateEnd),
                Map.entry("roomApplyTimeType", "0"),
                Map.entry("timeBegin", timeBegin),
                Map.entry("timeEnd", timeEnd)
        )));
        return jsonResponse("freeroom", freeRoomsJson(parseFreeRooms(html)));
    }

    public String getSemesters(String username, String password) throws Exception {
        return getSemesters(username, password, "", "");
    }

    public String getSemesters(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var page = get(client, config.baseUrl() + "eams/courseTableForStd.action");
        var params = extractCourseTableParams(page);
        return jsonResponse("semester", "[{\"SemesterID\":" + quote(params.semesterId()) + ",\"Ids\":" + quote(params.ids()) + ",\"Current\":true}]");
    }

    public String getCourse(String username, String password) throws Exception {
        return getCourse(username, password, "", "");
    }

    public String getCourse(String username, String password, String loginType, String authServerUrl) throws Exception {
        var cacheKey = valueOr(loginType, config.LoginType()) + ":" + authServerUrl + ":" + username;
        var cached = courseCache.get(cacheKey);
        if (cached != null && System.currentTimeMillis() - cached.createdAt < 3600_000) {
            return cached.json;
        }

        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var html = courseTableHtml(client);
        var teachers = parseTeachers(html);
        var courses = parseCourses(html);
        var json = jsonResponse("allcourse", coursesJson(courses));
        courseCache.put(cacheKey, new CacheItem(System.currentTimeMillis(), json, teachers, courses));
        return json;
    }

    public String getTeacher(String username, String password) throws Exception {
        return getTeacher(username, password, "", "");
    }

    public String getTeacher(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        return jsonResponse("teacher", teachersJson(parseTeachers(courseTableHtml(client))));
    }

    public String getWeekCourse(String username, String password, int week) throws Exception {
        return getWeekCourse(username, password, week, "", "");
    }

    public String getWeekCourse(String username, String password, int week, String loginType, String authServerUrl) throws Exception {
        getCourse(username, password, loginType, authServerUrl);
        var cacheKey = valueOr(loginType, config.LoginType()) + ":" + authServerUrl + ":" + username;
        var cached = courseCache.get(cacheKey);
        var result = new ArrayList<String>();
        for (var course : cached.courses) {
            if (week >= course.Weeks().length() || course.Weeks().charAt(week) != '1') {
                continue;
            }
            for (var teacher : cached.teachers) {
                if (!course.CourseID().contains(teacher.CourseID())) {
                    continue;
                }
                var times = new ArrayList<String>();
                for (var time : course.CourseTimes()) {
                    times.add(Integer.toString(time.TimeOfTheDay() + 1));
                }
                var day = course.CourseTimes().isEmpty() ? 0 : course.CourseTimes().get(0).DayOfTheWeek();
                result.add("{\"CourseName\":" + quote(teacher.CourseName()) + ",\"TeacherName\":" + quote(teacher.CourseTeacher()) + ",\"RoomName\":" + quote(course.RoomName()) + ",\"DayOfTheWeek\":" + day + ",\"TimeOfTheDay\":" + quote(String.join(",", times)) + "}");
            }
        }
        return jsonResponse("course", "[" + String.join(",", result) + "]");
    }

    public String getIcs(String username, String password) throws Exception {
        return getIcs(username, password, "", "");
    }

    public String getIcs(String username, String password, String loginType, String authServerUrl) throws Exception {
        getCourse(username, password, loginType, authServerUrl);
        var cacheKey = valueOr(loginType, config.LoginType()) + ":" + authServerUrl + ":" + username;
        var cached = courseCache.get(cacheKey);
        return jsonResponse("ics", quote(generateIcs(cached.courses)));
    }

    public String generateIcs(List<Course> courses) {
        var slots = config.ClassTimeSlots().isEmpty() ? DEFAULT_CLASS_TIME_SLOTS : config.ClassTimeSlots();
        var timezone = valueOr(config.CalendarTimezone(), "Asia/Shanghai");
        var calendarName = valueOr(config.CalendarName(), config.SchoolName() + "课表");
        var firstMonday = LocalDate.parse(config.CalendarFirst());
        var now = java.time.Instant.now().toString().replace("-", "").replace(":", "").replaceAll("\\.\\d+Z$", "Z");
        var builder = new StringBuilder();
        builder.append("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Ares-Gao//WeCourseService//CN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n");
        builder.append(foldIcsLine("X-WR-CALNAME:" + escapeIcsText(calendarName)));
        builder.append(foldIcsLine("X-WR-TIMEZONE:" + timezone));

        for (var course : courses) {
            var dayTimes = new HashMap<Integer, List<Integer>>();
            for (var time : course.CourseTimes()) {
                dayTimes.computeIfAbsent(time.DayOfTheWeek(), ignored -> new ArrayList<>()).add(time.TimeOfTheDay());
            }
            for (var entry : dayTimes.entrySet()) {
                var times = entry.getValue().stream().sorted().toList();
                var startSlot = times.get(0);
                var endSlot = times.get(times.size() - 1);
                if (startSlot >= slots.size() || endSlot >= slots.size()) {
                    continue;
                }
                for (int weekIndex = 0; weekIndex < course.Weeks().length(); weekIndex++) {
                    if (weekIndex == 0 || course.Weeks().charAt(weekIndex) != '1') {
                        continue;
                    }
                    var date = firstMonday.plusDays((weekIndex - 1) * 7L + entry.getKey());
                    var startAt = date.atTime(LocalTime.parse(slots.get(startSlot).Start())).format(DateTimeFormatter.ofPattern("yyyyMMdd'T'HHmmss"));
                    var endAt = date.atTime(LocalTime.parse(slots.get(endSlot).End())).format(DateTimeFormatter.ofPattern("yyyyMMdd'T'HHmmss"));
                    var uid = Pattern.compile("[^0-9A-Za-z_-]").matcher(course.CourseID() + "-" + weekIndex + "-" + entry.getKey() + "-" + startSlot).replaceAll("-") + "@wecourse.service";
                    builder.append("BEGIN:VEVENT\r\n");
                    builder.append(foldIcsLine("UID:" + uid));
                    builder.append("DTSTAMP:").append(now).append("\r\n");
                    builder.append(foldIcsLine("DTSTART;TZID=" + timezone + ":" + startAt));
                    builder.append(foldIcsLine("DTEND;TZID=" + timezone + ":" + endAt));
                    builder.append(foldIcsLine("SUMMARY:" + escapeIcsText(course.CourseName())));
                    builder.append(foldIcsLine("LOCATION:" + escapeIcsText(course.RoomName())));
                    builder.append(foldIcsLine("DESCRIPTION:" + escapeIcsText("CourseID: " + course.CourseID() + "\nRoomID: " + course.RoomID())));
                    builder.append("END:VEVENT\r\n");
                }
            }
        }
        builder.append("END:VCALENDAR\r\n");
        return builder.toString();
    }

    public String getAccount(String username, String password) throws Exception {
        return getAccount(username, password, "", "");
    }

    public String getAccount(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var html = get(client, config.baseUrl() + "eams/stdDetail.action");
        var info = matchAll(html, "(?i)<td>([^>]*)</td>");
        return jsonResponse("account", "{" +
                "\"FullName\":" + quote(info.get(0)) + "," +
                "\"EnglishName\":" + quote(info.get(1)) + "," +
                "\"Sex\":" + quote(info.get(2)) + "," +
                "\"StartTime\":" + quote(info.get(11)) + "," +
                "\"EndTime\":" + quote(info.get(12)) + "," +
                "\"SchoolYear\":" + quote(info.get(4)) + "," +
                "\"Type\":" + quote(info.get(5) + "(" + info.get(14) + ")") + "," +
                "\"System\":" + quote(info.get(8)) + "," +
                "\"Specialty\":" + quote(info.get(9)) + "," +
                "\"Class\":" + quote(info.get(18)) +
                "}");
    }

    public String getPhoto(String username, String password) throws Exception {
        return getPhoto(username, password, "", "");
    }

    public String getPhoto(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var request = HttpRequest.newBuilder(URI.create(config.baseUrl() + "eams/showSelfAvatar.action?user.name=" + URLEncoder.encode(username, StandardCharsets.UTF_8))).header("User-Agent", USER_AGENT).GET().build();
        var bytes = client.send(request, HttpResponse.BodyHandlers.ofByteArray()).body();
        return jsonResponse("photo", quote("data:image/jpg;base64," + Base64.getEncoder().encodeToString(bytes)));
    }

    public String getGrade(String username, String password) throws Exception {
        return getGrade(username, password, "", "");
    }

    public String getGrade(String username, String password, String loginType, String authServerUrl) throws Exception {
        var client = createLoggedInClient(username, password, loginType, authServerUrl);
        var html = post(client, config.baseUrl() + "eams/teach/grade/course/person!historyCourseGrade.action?projectType=MAJOR", "");
        var rows = matchRows(html);
        var grades = new ArrayList<String>();
        for (int i = 2; i < rows.size(); i++) {
            var cells = matchAll(rows.get(i), "(?i)<td.*>([^>]*)</td>");
            if (cells.size() < 6) {
                continue;
            }
            var sup = matchAll(rows.get(i), "(?i)<sup.*>([^>]*)</sup>");
            grades.add("{" +
                    "\"CourseID\":" + quote(cells.get(1).trim()) + "," +
                    "\"CourseName\":" + quote(sup.isEmpty() ? cells.get(3).trim() : sup.get(0)) + "," +
                    "\"CourseTerm\":" + quote(cells.get(0).trim()) + "," +
                    "\"CourseCredit\":" + quote(cells.get(4).trim()) + "," +
                    "\"CourseGrade\":" + quote(cells.get(cells.size() - 2).trim()) + "," +
                    "\"GradePoint\":" + quote(cells.get(cells.size() - 1).trim()) +
                    "}");
        }
        return jsonResponse("grade", "[" + String.join(",", grades) + "]");
    }

    private HttpClient createLoggedInClient(String username, String password, String loginType, String authServerUrl) throws Exception {
        var resolvedLoginType = valueOr(loginType, config.LoginType());
        if ("authserver".equalsIgnoreCase(resolvedLoginType)) {
            return createAuthServerLoggedInClient(username, password, authServerUrl);
        }

        var cookieManager = new CookieManager(null, CookiePolicy.ACCEPT_ALL);
        var client = HttpClient.newBuilder().cookieHandler(cookieManager).build();
        var loginHtml = get(client, config.baseUrl() + "eams/login.action");
        var salt = extractPasswordSalt(loginHtml);
        Thread.sleep(1000);
        var response = post(client, config.baseUrl() + "eams/login.action", form(Map.of(
                "username", username,
                "password", sha1(salt + password),
                "session_locale", "zh_CN")));
        if (!response.contains(LOGIN_MARKER)) {
            throw new IllegalStateException("Login failed.");
        }
        return client;
    }

    private HttpClient createAuthServerLoggedInClient(String username, String password, String authServerUrl) throws Exception {
        var loginUrl = valueOr(authServerUrl, config.AuthServerURL());
        if (loginUrl.isBlank()) {
            throw new IllegalStateException("AuthServerURL is required for authserver login.");
        }

        var cookieManager = new CookieManager(null, CookiePolicy.ACCEPT_ALL);
        var client = HttpClient.newBuilder().cookieHandler(cookieManager).build();
        var html = get(client, loginUrl);
        var salt = inputValue(html, "", "pwdEncryptSalt");
        var execution = inputValue(html, "execution", "");
        if (salt.isBlank() || execution.isBlank()) {
            throw new IllegalStateException("Authserver login page is missing pwdEncryptSalt or execution.");
        }

        var captcha = "";
        if (needAuthServerCaptcha(html)) {
            if (captchaSolver == null) {
                throw new IllegalStateException("Authserver captcha is required. Provide a CaptchaSolver implementation.");
            }
            var imageRequest = HttpRequest.newBuilder(URI.create(authServerBase(loginUrl) + "/getCaptcha.htl?" + System.currentTimeMillis())).header("User-Agent", USER_AGENT).GET().build();
            captcha = captchaSolver.solve(client.send(imageRequest, HttpResponse.BodyHandlers.ofByteArray()).body());
        }

        var response = post(client, loginUrl, form(Map.of(
                "username", username,
                "password", authServerEncryptPassword(password, salt),
                "captcha", captcha,
                "_eventId", valueOr(inputValue(html, "_eventId", ""), "submit"),
                "cllt", "userNameLogin",
                "dllt", valueOr(inputValue(html, "dllt", ""), "generalLogin"),
                "lt", inputValue(html, "lt", ""),
                "execution", execution,
                "rmShown", "1")));
        if (response.contains("认证失败")) {
            throw new IllegalStateException("Authserver login failed.");
        }
        return client;
    }

    private String courseTableHtml(HttpClient client) throws Exception {
        Thread.sleep(1000);
        var page = get(client, config.baseUrl() + "eams/courseTableForStd.action");
        var params = extractCourseTableParams(page);
        return post(client, config.baseUrl() + "eams/courseTableForStd!courseTable.action", form(Map.of(
                "ignoreHead", "1",
                "showPrintAndExport", "1",
                "setting.kind", "std",
                "startWeek", "",
                "semester.id", params.semesterId(),
                "ids", params.ids())));
    }

    private String teacherCourseTableHtml(HttpClient client) throws Exception {
        var page = get(client, config.baseUrl() + "eams/courseTableForTeacher.action");
        var ids = Pattern.compile("name=[\"']ids[\"'][^>]*value=[\"']([^\"']+)[\"']").matcher(page);
        var semester = Pattern.compile("semesterCalendar\\(\\{[^}]*value:[\"']([^\"']+)[\"']").matcher(page);
        if (!ids.find() || !semester.find()) {
            throw new IllegalStateException("Teacher course table params not found.");
        }
        return post(client, config.baseUrl() + "eams/courseTableForTeacher!courseTable.action", form(Map.of(
                "ignoreHead", "1",
                "setting.forSemester", "1",
                "ids", ids.group(1),
                "setting.kind", "teacher",
                "semester.id", semester.group(1)
        )));
    }

    private CourseTableParams extractCourseTableParams(String html) {
        var ids = Pattern.compile("bg\\.form\\.addInput\\(form,\\s*\"ids\",\\s*\"([^\"]+)\"").matcher(html);
        if (!ids.find()) {
            throw new IllegalStateException("Course table ids not found.");
        }

        var patterns = List.of(
                "name=[\"']semester\\.id[\"'][^>]*value=[\"']([^\"']+)[\"']",
                "semesterCalendar\\(\\{[^}]*value:\\s*\"([^\"]+)\"",
                "semesterCalendar\\(\\{[^}]*value:\\s*'([^']+)'",
                "bg\\.form\\.addInput\\(form,\\s*\"semester\\.id\",\\s*\"([^\"]+)\""
        );
        for (var pattern : patterns) {
            var semester = Pattern.compile(pattern, Pattern.CASE_INSENSITIVE).matcher(html);
            if (semester.find() && !semester.group(1).isBlank()) {
                return new CourseTableParams(ids.group(1), semester.group(1));
            }
        }

        throw new IllegalStateException("semester.id not found.");
    }

    private List<Teacher> parseTeachers(String html) {
        var rows = matchAllFull(html, "(?i)<td>(\\d)</td>\\s*<td>([:alpha:].+)</td>\\s*<td>(.+)</td>\\s*<td>((\\d)|(\\d\\.\\d))</td>\\s*<td>\\s*<a href=.*\\s.*\\s.*\\s.*>.*</a>\\s*</td>\\s*<td>(.*)</td>");
        var result = new ArrayList<Teacher>();
        for (var row : rows) {
            var cells = matchAll(row, "(?i)<td>([^>]*)</td>");
            var links = matchAll(row, "(?i)>([^>]*)</a>");
            if (cells.size() >= 5 && !links.isEmpty()) {
                result.add(new Teacher(links.get(0), cells.get(2), cells.get(3), cells.get(4)));
            }
        }
        return result;
    }

    private List<Course> parseCourses(String html) {
        var rows = Pattern.compile("TaskActivity\\(actTeacherId(?:\\.toString\\(\\)|.join\\(','\\)),[^,]*,\"(.*)\",\"(.*)\\(.*\\)\",\"(.*)\",\"(.*)\",\"(.*)\",null,null,assistantName,\"\",\"\"\\);((?:\\s*index =\\d+\\*unitCount\\+\\d+;\\s*.*\\s)+)").matcher(html);
        var result = new ArrayList<Course>();
        while (rows.find()) {
            var times = new ArrayList<CourseTime>();
            for (var indexText : rows.group(6).split("table0.activities\\[index]\\[table0.activities\\[index\\].length\\]=activity;")) {
                var index = Pattern.compile("\\s*index =(\\d+)\\*unitCount\\+(\\d+);\\s*").matcher(indexText);
                if (index.find()) {
                    times.add(new CourseTime(Integer.parseInt(index.group(1)), Integer.parseInt(index.group(2))));
                }
            }
            result.add(new Course(rows.group(1), cleanJsText(rows.group(2)), rows.group(3), rows.group(4), rows.group(5), times));
        }
        return result;
    }

    private static String cleanJsText(String text) {
        return text.replace("\"+periodInfo+\"", "").replace("\\\"", "\"");
    }

    private static Identity parseHomeExtIdentity(String html) {
        var category = Pattern.compile("<input[^>]+name=[\"']security\\.userCategoryId[\"'][^>]*value=[\"']([^\"']+)[\"']", Pattern.CASE_INSENSITIVE | Pattern.DOTALL).matcher(html);
        if (category.find()) {
            var id = category.group(1).trim();
            return switch (id) {
                case "1" -> new Identity("student", "学生", id);
                case "2" -> new Identity("teacher", "教师", id);
                default -> new Identity("unknown", "未知", id);
            };
        }
        if (html.contains("courseTableForStd.action") || html.contains("stdDetail.action") || html.contains("学生")) {
            return new Identity("student", "学生", "");
        }
        if (html.contains("courseTableForTeacher.action") || html.contains("teacherExamTable.action") || html.contains("教师")) {
            return new Identity("teacher", "教师", "");
        }
        return new Identity("unknown", "未知", "");
    }

    private static List<TeacherExam> parseTeacherExams(String html) {
        var result = new ArrayList<TeacherExam>();
        for (var section : html.split("(?=<div id=\\\"toolbar[^\\\"]*\\\")")) {
            var titleMatch = Pattern.compile("bg\\.ui\\.toolbar\\(\"[^\"]+\",'([^']*)'").matcher(section);
            var title = titleMatch.find() ? cleanHtmlCell(titleMatch.group(1)) : "";
            var headers = new ArrayList<String>();
            for (var row : examTableRows(section)) {
                var cells = row.cells();
                if (row.header()) {
                    headers = new ArrayList<>(cells);
                    continue;
                }
                if (cells.size() < 7 || cells.get(0).isBlank()) {
                    continue;
                }
                String studentCount = "";
                String chiefExaminer = "";
                String invigilators = "";
                String examTime = "";
                String examRoom = "";
                if (headers.size() >= cells.size()) {
                    studentCount = cellByHeader(cells, headers, "人数", "学生数");
                    chiefExaminer = cellByHeader(cells, headers, "主考");
                    invigilators = cellByHeader(cells, headers, "监考");
                    examTime = cellByHeader(cells, headers, "时间", "安排");
                    examRoom = cellByHeader(cells, headers, "地点", "考场", "教室");
                } else if (cells.size() >= 9) {
                    studentCount = cells.get(4);
                    chiefExaminer = cells.get(5);
                    invigilators = cells.get(6);
                    examTime = cells.get(7);
                    examRoom = cells.get(8);
                } else if (cells.size() >= 8) {
                    invigilators = cells.get(4);
                    studentCount = cells.get(5);
                    examTime = cells.get(6);
                    examRoom = cells.get(7);
                } else {
                    studentCount = cells.get(4);
                    examTime = cells.get(5);
                    examRoom = cells.get(6);
                }
                result.add(new TeacherExam(title, cells.get(0), cells.get(1), cells.get(2), cells.get(3), studentCount, chiefExaminer, invigilators, examTime, examRoom));
            }
        }
        return result;
    }

    private static List<TeacherExamBatch> parseTeacherExamBatches(String html) {
        var result = new ArrayList<TeacherExamBatch>();
        var matcher = Pattern.compile("(?s)<option\\s+value=[\"']([^\"']+)[\"']([^>]*)>(.*?)</option>", Pattern.CASE_INSENSITIVE).matcher(html);
        while (matcher.find()) {
            result.add(new TeacherExamBatch(matcher.group(1), cleanHtmlCell(matcher.group(3)), matcher.group(2).toLowerCase().contains("selected")));
        }
        return result;
    }

    private static List<FreeRoom> parseFreeRooms(String html) {
        var result = new ArrayList<FreeRoom>();
        for (var cells : tableRows(html)) {
            if (cells.size() >= 6 && !cells.get(0).isBlank()) {
                result.add(new FreeRoom(cells.get(0), cells.get(1), cells.get(2), cells.get(3), cells.get(4), cells.get(5)));
            }
        }
        return result;
    }

    private static List<List<String>> tableRows(String html) {
        var result = new ArrayList<List<String>>();
        var rows = Pattern.compile("(?s)<tr[^>]*>(.*?)</tr>").matcher(html);
        while (rows.find()) {
            var cells = new ArrayList<String>();
            var matcher = Pattern.compile("(?s)<td[^>]*>(.*?)</td>").matcher(rows.group(1));
            while (matcher.find()) {
                cells.add(cleanHtmlCell(matcher.group(1)));
            }
            if (!cells.isEmpty()) {
                result.add(cells);
            }
        }
        return result;
    }

    private record ExamTableRow(List<String> cells, boolean header) {
    }

    private static List<ExamTableRow> examTableRows(String html) {
        var result = new ArrayList<ExamTableRow>();
        var rows = Pattern.compile("(?s)<tr[^>]*>(.*?)</tr>").matcher(html);
        while (rows.find()) {
            var cells = new ArrayList<String>();
            var header = false;
            var matcher = Pattern.compile("(?s)<(td|th)[^>]*>(.*?)</(?:td|th)>", Pattern.CASE_INSENSITIVE).matcher(rows.group(1));
            while (matcher.find()) {
                if ("th".equalsIgnoreCase(matcher.group(1))) {
                    header = true;
                }
                cells.add(cleanHtmlCell(matcher.group(2)));
            }
            if (!cells.isEmpty()) {
                header = header || looksLikeExamHeader(cells);
                result.add(new ExamTableRow(cells, header));
            }
        }
        return result;
    }

    private static boolean looksLikeExamHeader(List<String> cells) {
        var headerNames = List.of("课程代码", "课程序号", "课程编号", "课程名称", "开课院系", "院系", "学分", "人数", "学生数", "主考", "监考", "考试时间", "考试地点", "地点", "考场", "教室");
        var count = 0;
        for (var cell : cells) {
            if (headerNames.contains(cell)) {
                count++;
            }
        }
        return count >= 3;
    }

    private static String cellByHeader(List<String> cells, List<String> headers, String... names) {
        for (var i = 0; i < headers.size() && i < cells.size(); i++) {
            for (var name : names) {
                if (headers.get(i).contains(name)) {
                    return cells.get(i);
                }
            }
        }
        return "";
    }

    private static String cleanHtmlCell(String value) {
        return Pattern.compile("(?s)<[^>]+>").matcher(value).replaceAll("").replace("&nbsp;", " ").trim().replaceAll("\\s+", " ");
    }

    private static String escapeIcsText(String value) {
        return value.replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "").replace(";", "\\;").replace(",", "\\,");
    }

    private static String foldIcsLine(String line) {
        if (line.length() <= 75) {
            return line + "\r\n";
        }
        var builder = new StringBuilder();
        for (int index = 0; index < line.length(); index += 75) {
            if (index > 0) {
                builder.append("\r\n ");
            }
            builder.append(line, index, Math.min(index + 75, line.length()));
        }
        builder.append("\r\n");
        return builder.toString();
    }

    private static String get(HttpClient client, String url) throws IOException, InterruptedException {
        var request = HttpRequest.newBuilder(URI.create(url)).header("User-Agent", USER_AGENT).GET().build();
        return client.send(request, HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8)).body();
    }

    private static String post(HttpClient client, String url, String body) throws IOException, InterruptedException {
        var request = HttpRequest.newBuilder(URI.create(url)).header("User-Agent", USER_AGENT).header("Content-Type", "application/x-www-form-urlencoded").POST(HttpRequest.BodyPublishers.ofString(body)).build();
        return client.send(request, HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8)).body();
    }

    private static String form(Map<String, String> form) {
        var parts = new ArrayList<String>();
        for (var item : form.entrySet()) {
            parts.add(URLEncoder.encode(item.getKey(), StandardCharsets.UTF_8) + "=" + URLEncoder.encode(item.getValue(), StandardCharsets.UTF_8));
        }
        return String.join("&", parts);
    }

    private static String extractPasswordSalt(String html) {
        var marker = "CryptoJS.SHA1(";
        var index = html.indexOf(marker);
        if (index < 0) {
            throw new IllegalStateException("Password salt not found.");
        }
        return html.substring(index + 15, index + 52);
    }

    private static String sha1(String value) throws Exception {
        var digest = MessageDigest.getInstance("SHA-1").digest(value.getBytes(StandardCharsets.UTF_8));
        var builder = new StringBuilder();
        for (byte item : digest) {
            builder.append(String.format("%02x", item));
        }
        return builder.toString();
    }

    private static String authServerEncryptPassword(String password, String salt) throws Exception {
        var cipher = Cipher.getInstance("AES/CBC/PKCS5Padding");
        cipher.init(
                Cipher.ENCRYPT_MODE,
                new SecretKeySpec(salt.trim().getBytes(StandardCharsets.UTF_8), "AES"),
                new IvParameterSpec(randomString(16).getBytes(StandardCharsets.UTF_8)));
        return Base64.getEncoder().encodeToString(cipher.doFinal((randomString(64) + password).getBytes(StandardCharsets.UTF_8)));
    }

    private static boolean needAuthServerCaptcha(String html) {
        return Pattern.compile("var\\s+_badCredentialsCount\\s*=\\s*\"0\"").matcher(html).find()
                || (html.contains("getCaptcha.htl") && html.contains("captchaDiv"));
    }

    private static String authServerBase(String loginUrl) {
        var uri = URI.create(loginUrl);
        var path = uri.getPath();
        var index = path.indexOf("/login");
        var context = index >= 0 ? path.substring(0, index) : "/authserver";
        return uri.getScheme() + "://" + uri.getAuthority() + context;
    }

    private static String randomString(int length) {
        var builder = new StringBuilder(length);
        for (int i = 0; i < length; i++) {
            builder.append(AES_CHARS.charAt(RANDOM.nextInt(AES_CHARS.length())));
        }
        return builder.toString();
    }

    private static String inputValue(String html, String name, String elementId) {
        var pattern = !elementId.isEmpty()
                ? "<input[^>]*id=[\"']" + Pattern.quote(elementId) + "[\"'][^>]*>"
                : "<input[^>]*name=[\"']" + Pattern.quote(name) + "[\"'][^>]*>";
        var input = Pattern.compile(pattern, Pattern.CASE_INSENSITIVE).matcher(html);
        if (!input.find()) {
            return "";
        }
        var value = Pattern.compile("value=[\"']([^\"']*)", Pattern.CASE_INSENSITIVE).matcher(input.group());
        return value.find() ? value.group(1) : "";
    }

    private static String valueOr(String value, String fallback) {
        return value == null || value.isEmpty() ? fallback : value;
    }

    private static List<String> matchAll(String text, String pattern) {
        var matcher = Pattern.compile(pattern).matcher(text);
        var result = new ArrayList<String>();
        while (matcher.find()) {
            result.add(matcher.group(1));
        }
        return result;
    }

    private static List<String> matchAllFull(String text, String pattern) {
        var matcher = Pattern.compile(pattern).matcher(text);
        var result = new ArrayList<String>();
        while (matcher.find()) {
            result.add(matcher.group(0));
        }
        return result;
    }

    private static List<String> matchRows(String text) {
        return matchAllFull(text, "(?i)<tr>[\\s\\S]*?</tr>");
    }

    private static String quote(String value) {
        return "\"" + value.replace("\\", "\\\\").replace("\"", "\\\"").replace("\n", "\\n").replace("\r", "\\r") + "\"";
    }

    private static String jsonResponse(String type, String dataJson) {
        return "{\"Type\":" + quote(type) + ",\"Data\":" + dataJson + "}";
    }

    private static String coursesJson(List<Course> courses) {
        var items = new ArrayList<String>();
        for (var course : courses) {
            var times = new ArrayList<String>();
            for (var time : course.CourseTimes()) {
                times.add("{\"DayOfTheWeek\":" + time.DayOfTheWeek() + ",\"TimeOfTheDay\":" + time.TimeOfTheDay() + "}");
            }
            items.add("{\"CourseID\":" + quote(course.CourseID()) + ",\"CourseName\":" + quote(course.CourseName()) + ",\"RoomID\":" + quote(course.RoomID()) + ",\"RoomName\":" + quote(course.RoomName()) + ",\"Weeks\":" + quote(course.Weeks()) + ",\"CourseTimes\":[" + String.join(",", times) + "]}");
        }
        return "[" + String.join(",", items) + "]";
    }

    private static String teachersJson(List<Teacher> teachers) {
        var items = new ArrayList<String>();
        for (var teacher : teachers) {
            items.add("{\"CourseID\":" + quote(teacher.CourseID()) + ",\"CourseName\":" + quote(teacher.CourseName()) + ",\"CourseCredit\":" + quote(teacher.CourseCredit()) + ",\"CourseTeacher\":" + quote(teacher.CourseTeacher()) + "}");
        }
        return "[" + String.join(",", items) + "]";
    }

    private static String teacherExamsJson(List<TeacherExam> exams) {
        var items = new ArrayList<String>();
        for (var exam : exams) {
            items.add("{\"Category\":" + quote(exam.Category()) + ",\"CourseID\":" + quote(exam.CourseID()) + ",\"CourseName\":" + quote(exam.CourseName()) + ",\"Department\":" + quote(exam.Department()) + ",\"Credit\":" + quote(exam.Credit()) + ",\"StudentCount\":" + quote(exam.StudentCount()) + ",\"ChiefExaminer\":" + quote(exam.ChiefExaminer()) + ",\"Invigilators\":" + quote(exam.Invigilators()) + ",\"ExamTime\":" + quote(exam.ExamTime()) + ",\"ExamRoom\":" + quote(exam.ExamRoom()) + "}");
        }
        return "[" + String.join(",", items) + "]";
    }

    private static String teacherExamBatchesJson(List<TeacherExamBatch> batches) {
        var items = new ArrayList<String>();
        for (var batch : batches) {
            items.add("{\"ExamBatchID\":" + quote(batch.ExamBatchID()) + ",\"Name\":" + quote(batch.Name()) + ",\"Selected\":" + batch.Selected() + "}");
        }
        return "[" + String.join(",", items) + "]";
    }

    private static String freeRoomsJson(List<FreeRoom> rooms) {
        var items = new ArrayList<String>();
        for (var room : rooms) {
            items.add("{\"Index\":" + quote(room.Index()) + ",\"Name\":" + quote(room.Name()) + ",\"Building\":" + quote(room.Building()) + ",\"Campus\":" + quote(room.Campus()) + ",\"TypeName\":" + quote(room.TypeName()) + ",\"Capacity\":" + quote(room.Capacity()) + "}");
        }
        return "[" + String.join(",", items) + "]";
    }

    private record CacheItem(long createdAt, String json, List<Teacher> teachers, List<Course> courses) {
    }

    private record CourseTableParams(String ids, String semesterId) {
    }
}
