<?php

final class SupwisdomClient
{
    private const USER_AGENT = 'Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0';
    private const LOGIN_MARKER = '<a href="/eams/security/my.action" target="_blank" title="查看详情" style="color:#ffffff">';
    private const AES_CHARS = 'ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678';

    /** @var array<string, array{time:int,json:string,teachers:array<int,array<string,string>>}> */
    private array $courseCache = [];

    public function __construct(
        private readonly WeCourseConfig $config,
        private readonly mixed $captchaSolver = null,
    )
    {
    }

    public function login(string $username, string $password): string
    {
        try {
            $cookie = $this->createLoggedInCookie($username, $password);
            @unlink($cookie);
            return $this->jsonResponse('login', '登录成功');
        } catch (Throwable) {
            return $this->jsonResponse('login', '登录失败');
        }
    }

    public function getWeekTime(): string
    {
        $start = strtotime($this->config->CalendarFirst . ' 00:00:00');
        $week = (int) round((time() - $start) / 60 / 60 / 24 / 7) + 1;
        return $this->jsonResponse('week', (string) $week);
    }

    public function getSemesters(string $username, string $password): string
    {
        $cookie = $this->createLoggedInCookie($username, $password);
        try {
            $page = $this->request('GET', $this->config->baseUrl() . 'eams/courseTableForStd.action', [], $cookie);
            [$ids, $semesterId] = $this->extractCourseTableParams($page);
            return $this->jsonResponse('semester', [
                ['SemesterID' => $semesterId, 'Ids' => $ids, 'Current' => true],
            ]);
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getCourse(string $username, string $password): string
    {
        if (isset($this->courseCache[$username]) && time() - $this->courseCache[$username]['time'] < 3600) {
            return $this->courseCache[$username]['json'];
        }

        $cookie = $this->createLoggedInCookie($username, $password);
        try {
            $html = $this->courseTableHtml($cookie);
            $teachers = $this->parseTeachers($html);
            $courses = $this->parseCourses($html);
            $json = $this->jsonResponse('allcourse', $courses);
            $this->courseCache[$username] = ['time' => time(), 'json' => $json, 'teachers' => $teachers];
            return $json;
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getTeacher(string $username, string $password): string
    {
        $cookie = $this->createLoggedInCookie($username, $password);
        try {
            return $this->jsonResponse('teacher', $this->parseTeachers($this->courseTableHtml($cookie)));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getWeekCourse(string $username, string $password, int $week): string
    {
        $payload = json_decode($this->getCourse($username, $password), true, 512, JSON_THROW_ON_ERROR);
        $teachers = $this->courseCache[$username]['teachers'] ?? [];
        $result = [];

        foreach ($payload['Data'] as $course) {
            if ($week >= strlen($course['Weeks']) || $course['Weeks'][$week] !== '1') {
                continue;
            }
            foreach ($teachers as $teacher) {
                if (!str_contains($course['CourseID'], $teacher['CourseID'])) {
                    continue;
                }
                $times = implode(',', array_map(fn ($item) => (string) ($item['TimeOfTheDay'] + 1), $course['CourseTimes']));
                $result[] = [
                    'CourseName' => $teacher['CourseName'],
                    'TeacherName' => $teacher['CourseTeacher'],
                    'RoomName' => $course['RoomName'],
                    'DayOfTheWeek' => $course['CourseTimes'][0]['DayOfTheWeek'] ?? 0,
                    'TimeOfTheDay' => $times,
                ];
            }
        }

        return $this->jsonResponse('course', $result);
    }

    public function getAccount(string $username, string $password): string
    {
        $cookie = $this->createLoggedInCookie($username, $password);
        try {
            $html = $this->request('GET', $this->config->baseUrl() . 'eams/stdDetail.action', [], $cookie);
            preg_match_all('/(?i)<td>([^>]*)<\/td>/', $html, $matches);
            $info = $matches[1];
            return $this->jsonResponse('account', [
                'FullName' => $info[0],
                'EnglishName' => $info[1],
                'Sex' => $info[2],
                'StartTime' => $info[11],
                'EndTime' => $info[12],
                'SchoolYear' => $info[4],
                'Type' => $info[5] . '(' . $info[14] . ')',
                'System' => $info[8],
                'Specialty' => $info[9],
                'Class' => $info[18],
            ]);
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getPhoto(string $username, string $password): string
    {
        $cookie = $this->createLoggedInCookie($username, $password);
        try {
            $image = $this->request('GET', $this->config->baseUrl() . 'eams/showSelfAvatar.action?user.name=' . urlencode($username), [], $cookie);
            return $this->jsonResponse('photo', 'data:image/jpg;base64,' . base64_encode($image));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getGrade(string $username, string $password): string
    {
        $cookie = $this->createLoggedInCookie($username, $password);
        try {
            $html = $this->request('POST', $this->config->baseUrl() . 'eams/teach/grade/course/person!historyCourseGrade.action?projectType=MAJOR', [], $cookie);
            preg_match_all('/(?i)<tr>[\s\S]*?<\/tr>/', $html, $rows);
            $grades = [];
            foreach (array_slice($rows[0], 2) as $row) {
                preg_match_all('/(?i)<td.*>([^>]*)<\/td>/', $row, $cells);
                if (count($cells[1]) < 6) {
                    continue;
                }
                preg_match('/(?i)<sup.*>([^>]*)<\/sup>/', $row, $sup);
                $data = $cells[1];
                $grades[] = [
                    'CourseID' => trim($data[1], "\n"),
                    'CourseName' => $sup[1] ?? trim($data[3], "\t\r\n"),
                    'CourseTerm' => trim($data[0], "\n"),
                    'CourseCredit' => trim($data[4], "\n"),
                    'CourseGrade' => trim($data[count($data) - 2], "\t\n"),
                    'GradePoint' => trim($data[count($data) - 1], "\t\n"),
                ];
            }
            return $this->jsonResponse('grade', $grades);
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    private function createLoggedInCookie(string $username, string $password): string
    {
        if (strtolower($this->config->LoginType) === 'authserver') {
            return $this->createAuthServerLoggedInCookie($username, $password);
        }

        $cookie = tempnam(sys_get_temp_dir(), 'wecourse_cookie_');
        $loginHtml = $this->request('GET', $this->config->baseUrl() . 'eams/login.action', [], $cookie);
        $salt = $this->extractPasswordSalt($loginHtml);
        $hashedPassword = sha1($salt . $password);
        sleep(1);
        $response = $this->request('POST', $this->config->baseUrl() . 'eams/login.action', [
            'username' => $username,
            'password' => $hashedPassword,
            'session_locale' => 'zh_CN',
        ], $cookie);
        if (!str_contains($response, self::LOGIN_MARKER)) {
            throw new RuntimeException('Login failed.');
        }
        return $cookie;
    }

    private function createAuthServerLoggedInCookie(string $username, string $password): string
    {
        if ($this->config->AuthServerURL === '') {
            throw new RuntimeException('AuthServerURL is required for authserver login.');
        }

        $cookie = tempnam(sys_get_temp_dir(), 'wecourse_cookie_');
        $html = $this->request('GET', $this->config->AuthServerURL, [], $cookie);
        $salt = $this->inputValue($html, elementId: 'pwdEncryptSalt');
        $execution = $this->inputValue($html, name: 'execution');
        if ($salt === '' || $execution === '') {
            throw new RuntimeException('Authserver login page is missing pwdEncryptSalt or execution.');
        }

        $captcha = '';
        if ($this->needAuthServerCaptcha($html)) {
            if (!is_callable($this->captchaSolver)) {
                throw new RuntimeException('Authserver captcha is required. Provide a callable captcha solver.');
            }
            $image = $this->request('GET', $this->authServerBase() . '/getCaptcha.htl?' . time(), [], $cookie);
            $captcha = (string) call_user_func($this->captchaSolver, $image);
        }

        $response = $this->request('POST', $this->config->AuthServerURL, [
            'username' => $username,
            'password' => $this->authServerEncryptPassword($password, $salt),
            'captcha' => $captcha,
            '_eventId' => $this->inputValue($html, name: '_eventId') ?: 'submit',
            'cllt' => 'userNameLogin',
            'dllt' => $this->inputValue($html, name: 'dllt') ?: 'generalLogin',
            'lt' => $this->inputValue($html, name: 'lt'),
            'execution' => $execution,
            'rmShown' => '1',
        ], $cookie);
        if (str_contains($response, '认证失败')) {
            throw new RuntimeException('Authserver login failed.');
        }
        return $cookie;
    }

    private function courseTableHtml(string $cookie): string
    {
        sleep(1);
        $page = $this->request('GET', $this->config->baseUrl() . 'eams/courseTableForStd.action', [], $cookie);
        [$ids, $semesterId] = $this->extractCourseTableParams($page);
        return $this->request('POST', $this->config->baseUrl() . 'eams/courseTableForStd!courseTable.action', [
            'ignoreHead' => '1',
            'showPrintAndExport' => '1',
            'setting.kind' => 'std',
            'startWeek' => '',
            'semester.id' => $semesterId,
            'ids' => $ids,
        ], $cookie);
    }

    /** @return array{0:string,1:string} */
    private function extractCourseTableParams(string $html): array
    {
        if (!preg_match('/bg\.form\.addInput\(form,\s*"ids",\s*"([^"]+)"/', $html, $ids)) {
            throw new RuntimeException('Course table ids not found.');
        }

        $patterns = [
            '/name=["\']semester\.id["\'][^>]*value=["\']([^"\']+)["\']/i',
            '/semesterCalendar\(\{[^}]*value:\s*"([^"]+)"/i',
            "/semesterCalendar\(\{[^}]*value:\s*'([^']+)'/i",
            '/bg\.form\.addInput\(form,\s*"semester\.id",\s*"([^"]+)"/i',
        ];
        foreach ($patterns as $pattern) {
            if (preg_match($pattern, $html, $semester) && $semester[1] !== '') {
                return [$ids[1], $semester[1]];
            }
        }

        throw new RuntimeException('semester.id not found.');
    }

    /** @return array<int,array<string,string>> */
    private function parseTeachers(string $html): array
    {
        preg_match_all('/(?i)<td>(\d)<\/td>\s*<td>([:alpha:].+)<\/td>\s*<td>(.+)<\/td>\s*<td>((\d)|(\d\.\d))<\/td>\s*<td>\s*<a href=.*\s.*\s.*\s.*>.*<\/a>\s*<\/td>\s*<td>(.*)<\/td>/', $html, $rows);
        $teachers = [];
        foreach ($rows[0] as $row) {
            preg_match_all('/(?i)<td>([^>]*)<\/td>/', $row, $cells);
            preg_match_all('/(?i)>([^>]*)<\/a>/', $row, $links);
            if (count($cells[1]) >= 5 && count($links[1]) > 0) {
                $teachers[] = [
                    'CourseID' => $links[1][0],
                    'CourseName' => $cells[1][2],
                    'CourseCredit' => $cells[1][3],
                    'CourseTeacher' => $cells[1][4],
                ];
            }
        }
        return $teachers;
    }

    /** @return array<int,array<string,mixed>> */
    private function parseCourses(string $html): array
    {
        preg_match_all('/TaskActivity\(actTeacherId.join\(\'\,\'\),actTeacherName.join\(\'\,\'\),"(.+)","(.+)\(.*\)","(.+)","(.+)","(.+)",null,null,assistantName,"",""\);((?:\s*index =\d+\*unitCount\+\d+;\s*.*\s)+)/', $html, $rows, PREG_SET_ORDER);
        $courses = [];
        foreach ($rows as $row) {
            $times = [];
            foreach (explode('table0.activities[index][table0.activities[index].length]=activity;', $row[6]) as $indexText) {
                if (preg_match('/\s*index =(\d+)\*unitCount\+(\d+);\s*/', $indexText, $index)) {
                    $times[] = ['DayOfTheWeek' => (int) $index[1], 'TimeOfTheDay' => (int) $index[2]];
                }
            }
            $courses[] = [
                'CourseID' => $row[1],
                'CourseName' => $this->cleanJsText($row[2]),
                'RoomID' => $row[3],
                'RoomName' => $row[4],
                'Weeks' => $row[5],
                'CourseTimes' => $times,
            ];
        }
        return $courses;
    }

    /** @param array<string,string> $form */
    private function request(string $method, string $url, array $form = [], ?string $cookie = null): string
    {
        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_USERAGENT => self::USER_AGENT,
            CURLOPT_TIMEOUT => 20,
        ]);
        if ($cookie !== null) {
            curl_setopt($ch, CURLOPT_COOKIEJAR, $cookie);
            curl_setopt($ch, CURLOPT_COOKIEFILE, $cookie);
        }
        if ($method === 'POST') {
            curl_setopt($ch, CURLOPT_POST, true);
            curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($form));
            curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/x-www-form-urlencoded']);
        }
        $response = curl_exec($ch);
        if ($response === false) {
            throw new RuntimeException(curl_error($ch));
        }
        curl_close($ch);
        return $response;
    }

    private function cleanJsText(string $text): string
    {
        return str_replace(['"+periodInfo+"', '\\"'], ['', '"'], $text);
    }

    private function extractPasswordSalt(string $html): string
    {
        $marker = 'CryptoJS.SHA1(';
        $index = strpos($html, $marker);
        if ($index === false) {
            throw new RuntimeException('Password salt not found.');
        }
        return substr($html, $index + 15, 37);
    }

    private function authServerEncryptPassword(string $password, string $salt): string
    {
        $payload = $this->randomString(64) . $password;
        $iv = $this->randomString(16);
        $encrypted = openssl_encrypt($payload, 'AES-128-CBC', trim($salt), OPENSSL_RAW_DATA, $iv);
        if ($encrypted === false) {
            throw new RuntimeException('Authserver password encryption failed.');
        }
        return base64_encode($encrypted);
    }

    private function needAuthServerCaptcha(string $html): bool
    {
        return preg_match('/var\s+_badCredentialsCount\s*=\s*"0"/', $html) === 1
            || (str_contains($html, 'getCaptcha.htl') && str_contains($html, 'captchaDiv'));
    }

    private function authServerBase(): string
    {
        $parts = parse_url($this->config->AuthServerURL);
        $path = $parts['path'] ?? '/authserver/login';
        $index = strpos($path, '/login');
        $context = $index === false ? '/authserver' : substr($path, 0, $index);
        return ($parts['scheme'] ?? 'https') . '://' . ($parts['host'] ?? '') . $context;
    }

    private function randomString(int $length): string
    {
        $result = '';
        $max = strlen(self::AES_CHARS) - 1;
        for ($i = 0; $i < $length; $i++) {
            $result .= self::AES_CHARS[random_int(0, $max)];
        }
        return $result;
    }

    private function inputValue(string $html, string $name = '', string $elementId = ''): string
    {
        $pattern = $elementId !== ''
            ? '/<input[^>]*id=["\']' . preg_quote($elementId, '/') . '["\'][^>]*>/i'
            : '/<input[^>]*name=["\']' . preg_quote($name, '/') . '["\'][^>]*>/i';
        if (!preg_match($pattern, $html, $input)) {
            return '';
        }
        return preg_match('/value=["\']([^"\']*)/i', $input[0], $value) ? $value[1] : '';
    }

    private function jsonResponse(string $type, mixed $data): string
    {
        return json_encode(['Type' => $type, 'Data' => $data], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_THROW_ON_ERROR);
    }
}
