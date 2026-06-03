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
import java.time.LocalDateTime;
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

    public SupwisdomClient(WeCourseConfig config) {
        this(config, null);
    }

    public SupwisdomClient(WeCourseConfig config, CaptchaSolver captchaSolver) {
        this.config = config;
        this.captchaSolver = captchaSolver;
    }

    public String login(String username, String password) {
        try {
            createLoggedInClient(username, password);
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

    public String getSemesters(String username, String password) throws Exception {
        var client = createLoggedInClient(username, password);
        var page = get(client, config.baseUrl() + "eams/courseTableForStd.action");
        var params = extractCourseTableParams(page);
        return jsonResponse("semester", "[{\"SemesterID\":" + quote(params.semesterId()) + ",\"Ids\":" + quote(params.ids()) + ",\"Current\":true}]");
    }

    public String getCourse(String username, String password) throws Exception {
        var cached = courseCache.get(username);
        if (cached != null && System.currentTimeMillis() - cached.createdAt < 3600_000) {
            return cached.json;
        }

        var client = createLoggedInClient(username, password);
        var html = courseTableHtml(client);
        var teachers = parseTeachers(html);
        var courses = parseCourses(html);
        var json = jsonResponse("allcourse", coursesJson(courses));
        courseCache.put(username, new CacheItem(System.currentTimeMillis(), json, teachers, courses));
        return json;
    }

    public String getTeacher(String username, String password) throws Exception {
        var client = createLoggedInClient(username, password);
        return jsonResponse("teacher", teachersJson(parseTeachers(courseTableHtml(client))));
    }

    public String getWeekCourse(String username, String password, int week) throws Exception {
        getCourse(username, password);
        var cached = courseCache.get(username);
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

    public String getAccount(String username, String password) throws Exception {
        var client = createLoggedInClient(username, password);
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
        var client = createLoggedInClient(username, password);
        var request = HttpRequest.newBuilder(URI.create(config.baseUrl() + "eams/showSelfAvatar.action?user.name=" + URLEncoder.encode(username, StandardCharsets.UTF_8))).header("User-Agent", USER_AGENT).GET().build();
        var bytes = client.send(request, HttpResponse.BodyHandlers.ofByteArray()).body();
        return jsonResponse("photo", quote("data:image/jpg;base64," + Base64.getEncoder().encodeToString(bytes)));
    }

    public String getGrade(String username, String password) throws Exception {
        var client = createLoggedInClient(username, password);
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

    private HttpClient createLoggedInClient(String username, String password) throws Exception {
        if ("authserver".equalsIgnoreCase(config.LoginType())) {
            return createAuthServerLoggedInClient(username, password);
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

    private HttpClient createAuthServerLoggedInClient(String username, String password) throws Exception {
        if (config.AuthServerURL().isBlank()) {
            throw new IllegalStateException("AuthServerURL is required for authserver login.");
        }

        var cookieManager = new CookieManager(null, CookiePolicy.ACCEPT_ALL);
        var client = HttpClient.newBuilder().cookieHandler(cookieManager).build();
        var html = get(client, config.AuthServerURL());
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
            var imageRequest = HttpRequest.newBuilder(URI.create(authServerBase(config.AuthServerURL()) + "/getCaptcha.htl?" + System.currentTimeMillis())).header("User-Agent", USER_AGENT).GET().build();
            captcha = captchaSolver.solve(client.send(imageRequest, HttpResponse.BodyHandlers.ofByteArray()).body());
        }

        var response = post(client, config.AuthServerURL(), form(Map.of(
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
        var rows = Pattern.compile("TaskActivity\\(actTeacherId.join\\(','\\),actTeacherName.join\\(','\\),\"(.*)\",\"(.*)\\(.*\\)\",\"(.*)\",\"(.*)\",\"(.*)\",null,null,assistantName,\"\",\"\"\\);((?:\\s*index =\\d+\\*unitCount\\+\\d+;\\s*.*\\s)+)").matcher(html);
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

    private record CacheItem(long createdAt, String json, List<Teacher> teachers, List<Course> courses) {
    }

    private record CourseTableParams(String ids, String semesterId) {
    }
}
