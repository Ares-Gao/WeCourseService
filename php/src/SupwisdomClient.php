<?php

final class SupwisdomClient
{
    private const USER_AGENT = 'Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0';
    private const LOGIN_MARKER = '<a href="/eams/security/my.action" target="_blank" title="查看详情" style="color:#ffffff">';
    private const AES_CHARS = 'ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678';
    private const DEFAULT_CLASS_TIME_SLOTS = [
        ['Start' => '08:00', 'End' => '08:45'],
        ['Start' => '08:55', 'End' => '09:40'],
        ['Start' => '10:00', 'End' => '10:45'],
        ['Start' => '10:55', 'End' => '11:40'],
        ['Start' => '14:00', 'End' => '14:45'],
        ['Start' => '14:55', 'End' => '15:40'],
        ['Start' => '16:00', 'End' => '16:45'],
        ['Start' => '16:55', 'End' => '17:40'],
        ['Start' => '19:00', 'End' => '19:45'],
        ['Start' => '19:55', 'End' => '20:40'],
        ['Start' => '20:50', 'End' => '21:35'],
        ['Start' => '21:45', 'End' => '22:30'],
    ];

    /** @var array<string, array{time:int,json:string,teachers:array<int,array<string,string>>}> */
    private array $courseCache = [];

    public function __construct(
        private readonly WeCourseConfig $config,
        private readonly mixed $captchaSolver = null,
    )
    {
    }

    public function login(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        try {
            $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
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

    public function getIdentity(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
        try {
            $html = $this->request('GET', $this->config->baseUrl() . 'eams/homeExt.action', [], $cookie);
            return $this->jsonResponse('identity', $this->parseHomeExtIdentity($html));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getTeacherCourse(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
        try {
            return $this->jsonResponse('teachercourse', $this->parseCourses($this->teacherCourseTableHtml($cookie)));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getTeacherExam(string $username, string $password, string $loginType = '', string $authServerUrl = '', string $examBatchId = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
        try {
            $page = $this->request('GET', $this->config->baseUrl() . 'eams/teacherExamTable.action', [], $cookie);
            if ($examBatchId === '' && preg_match('/<option value=["\']([^"\']+)["\'][^>]*selected/i', $page, $match)) {
                $examBatchId = $match[1];
            }
            $examBatchId = $examBatchId !== '' ? $examBatchId : '601';
            $html = $this->request('GET', $this->config->baseUrl() . 'eams/teacherExamTable!examAtivities.action?examBatch.id=' . rawurlencode($examBatchId), [], $cookie);
            return $this->jsonResponse('teacherexam', $this->parseTeacherExams($html));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getFreeRoom(string $dateBegin, string $dateEnd = '', string $timeBegin = '1', string $timeEnd = ''): string
    {
        $dateBegin = $dateBegin !== '' ? $dateBegin : date('Y-m-d');
        $dateEnd = $dateEnd !== '' ? $dateEnd : $dateBegin;
        $timeEnd = $timeEnd !== '' ? $timeEnd : $timeBegin;
        $html = $this->request('POST', $this->config->baseUrl() . 'eams/publicFree!search.action', [
            'classroom.type.id' => '',
            'classroom.campus.id' => '',
            'classroom.building.id' => '',
            'seats' => '',
            'classroom.name' => '',
            'cycleTime.cycleCount' => '1',
            'cycleTime.cycleType' => '1',
            'cycleTime.dateBegin' => $dateBegin,
            'cycleTime.dateEnd' => $dateEnd,
            'roomApplyTimeType' => '0',
            'timeBegin' => $timeBegin,
            'timeEnd' => $timeEnd,
        ]);
        return $this->jsonResponse('freeroom', $this->parseFreeRooms($html));
    }

    public function getSemesters(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
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

    public function getCourse(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cacheKey = ($loginType !== '' ? $loginType : $this->config->LoginType) . ':' . $authServerUrl . ':' . $username;
        if (isset($this->courseCache[$cacheKey]) && time() - $this->courseCache[$cacheKey]['time'] < 3600) {
            return $this->courseCache[$cacheKey]['json'];
        }

        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
        try {
            $html = $this->courseTableHtml($cookie);
            $teachers = $this->parseTeachers($html);
            $courses = $this->parseCourses($html);
            $json = $this->jsonResponse('allcourse', $courses);
            $this->courseCache[$cacheKey] = ['time' => time(), 'json' => $json, 'teachers' => $teachers];
            return $json;
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getTeacher(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
        try {
            return $this->jsonResponse('teacher', $this->parseTeachers($this->courseTableHtml($cookie)));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getWeekCourse(string $username, string $password, int $week, string $loginType = '', string $authServerUrl = ''): string
    {
        $payload = json_decode($this->getCourse($username, $password, $loginType, $authServerUrl), true, 512, JSON_THROW_ON_ERROR);
        $cacheKey = ($loginType !== '' ? $loginType : $this->config->LoginType) . ':' . $authServerUrl . ':' . $username;
        $teachers = $this->courseCache[$cacheKey]['teachers'] ?? [];
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

    public function getIcs(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $payload = json_decode($this->getCourse($username, $password, $loginType, $authServerUrl), true, 512, JSON_THROW_ON_ERROR);
        return $this->jsonResponse('ics', $this->generateIcs($payload['Data'] ?? []));
    }

    /** @param array<int,array<string,mixed>> $courses */
    public function generateIcs(array $courses): string
    {
        $slots = $this->config->ClassTimeSlots ?: self::DEFAULT_CLASS_TIME_SLOTS;
        $timezone = $this->config->CalendarTimezone ?: 'Asia/Shanghai';
        $calendarName = $this->config->CalendarName ?: $this->config->SchoolName . '课表';
        $firstMonday = strtotime($this->config->CalendarFirst . ' 00:00:00');
        $now = gmdate('Ymd\THis\Z');
        $ics = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Ares-Gao//WeCourseService//CN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n";
        $ics .= $this->foldIcsLine('X-WR-CALNAME:' . $this->escapeIcsText($calendarName));
        $ics .= $this->foldIcsLine('X-WR-TIMEZONE:' . $timezone);

        foreach ($courses as $course) {
            $dayTimes = [];
            foreach ($course['CourseTimes'] ?? [] as $time) {
                $dayTimes[(int) $time['DayOfTheWeek']][] = (int) $time['TimeOfTheDay'];
            }
            foreach ($dayTimes as $day => $times) {
                sort($times);
                $startSlot = $times[0];
                $endSlot = $times[count($times) - 1];
                if (!isset($slots[$startSlot], $slots[$endSlot])) {
                    continue;
                }
                foreach (str_split((string) ($course['Weeks'] ?? '')) as $weekIndex => $enabled) {
                    if ($weekIndex === 0 || $enabled !== '1') {
                        continue;
                    }
                    $date = strtotime('+' . (($weekIndex - 1) * 7 + $day) . ' days', $firstMonday);
                    $startAt = date('Ymd', $date) . 'T' . str_replace(':', '', $slots[$startSlot]['Start']) . '00';
                    $endAt = date('Ymd', $date) . 'T' . str_replace(':', '', $slots[$endSlot]['End']) . '00';
                    $uid = preg_replace('/[^0-9A-Za-z_-]/', '-', ($course['CourseID'] ?? '') . '-' . $weekIndex . '-' . $day . '-' . $startSlot) . '@wecourse.service';

                    $ics .= "BEGIN:VEVENT\r\n";
                    $ics .= $this->foldIcsLine('UID:' . $uid);
                    $ics .= 'DTSTAMP:' . $now . "\r\n";
                    $ics .= $this->foldIcsLine('DTSTART;TZID=' . $timezone . ':' . $startAt);
                    $ics .= $this->foldIcsLine('DTEND;TZID=' . $timezone . ':' . $endAt);
                    $ics .= $this->foldIcsLine('SUMMARY:' . $this->escapeIcsText((string) ($course['CourseName'] ?? '')));
                    $ics .= $this->foldIcsLine('LOCATION:' . $this->escapeIcsText((string) ($course['RoomName'] ?? '')));
                    $ics .= $this->foldIcsLine('DESCRIPTION:' . $this->escapeIcsText('CourseID: ' . ($course['CourseID'] ?? '') . "\nRoomID: " . ($course['RoomID'] ?? '')));
                    $ics .= "END:VEVENT\r\n";
                }
            }
        }
        return $ics . "END:VCALENDAR\r\n";
    }

    public function getAccount(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
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

    public function getPhoto(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
        try {
            $image = $this->request('GET', $this->config->baseUrl() . 'eams/showSelfAvatar.action?user.name=' . urlencode($username), [], $cookie);
            return $this->jsonResponse('photo', 'data:image/jpg;base64,' . base64_encode($image));
        } finally {
            $this->request('GET', $this->config->baseUrl() . 'eams/logout.action', [], $cookie);
            @unlink($cookie);
        }
    }

    public function getGrade(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $cookie = $this->createLoggedInCookie($username, $password, $loginType, $authServerUrl);
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

    private function createLoggedInCookie(string $username, string $password, string $loginType = '', string $authServerUrl = ''): string
    {
        $resolvedLoginType = $loginType !== '' ? $loginType : $this->config->LoginType;
        if (strtolower($resolvedLoginType) === 'authserver') {
            return $this->createAuthServerLoggedInCookie($username, $password, $authServerUrl);
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

    private function createAuthServerLoggedInCookie(string $username, string $password, string $authServerUrl = ''): string
    {
        $loginUrl = $authServerUrl !== '' ? $authServerUrl : $this->config->AuthServerURL;
        if ($loginUrl === '') {
            throw new RuntimeException('AuthServerURL is required for authserver login.');
        }

        $cookie = tempnam(sys_get_temp_dir(), 'wecourse_cookie_');
        $html = $this->request('GET', $loginUrl, [], $cookie);
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
            $image = $this->request('GET', $this->authServerBase($loginUrl) . '/getCaptcha.htl?' . time(), [], $cookie);
            $captcha = (string) call_user_func($this->captchaSolver, $image);
        }

        $response = $this->request('POST', $loginUrl, [
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

    private function teacherCourseTableHtml(string $cookie): string
    {
        $page = $this->request('GET', $this->config->baseUrl() . 'eams/courseTableForTeacher.action', [], $cookie);
        if (!preg_match('/name=["\']ids["\'][^>]*value=["\']([^"\']+)["\']/', $page, $ids)
            || !preg_match('/semesterCalendar\(\{[^}]*value:["\']([^"\']+)["\']/', $page, $semester)) {
            throw new RuntimeException('Teacher course table params not found.');
        }
        return $this->request('POST', $this->config->baseUrl() . 'eams/courseTableForTeacher!courseTable.action', [
            'ignoreHead' => '1',
            'setting.forSemester' => '1',
            'ids' => $ids[1],
            'setting.kind' => 'teacher',
            'semester.id' => $semester[1],
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

    private function parseHomeExtIdentity(string $html): array
    {
        if (preg_match('/<input[^>]+name=["\']security\.userCategoryId["\'][^>]*value=["\']([^"\']+)["\']/is', $html, $match)) {
            $category = trim($match[1]);
            return match ($category) {
                '1' => ['Role' => 'student', 'RoleName' => '学生', 'UserCategoryID' => $category],
                '2' => ['Role' => 'teacher', 'RoleName' => '教师', 'UserCategoryID' => $category],
                default => ['Role' => 'unknown', 'RoleName' => '未知', 'UserCategoryID' => $category],
            };
        }
        if (str_contains($html, 'courseTableForStd.action') || str_contains($html, 'stdDetail.action') || str_contains($html, '学生')) {
            return ['Role' => 'student', 'RoleName' => '学生', 'UserCategoryID' => ''];
        }
        if (str_contains($html, 'courseTableForTeacher.action') || str_contains($html, 'teacherExamTable.action') || str_contains($html, '教师')) {
            return ['Role' => 'teacher', 'RoleName' => '教师', 'UserCategoryID' => ''];
        }
        return ['Role' => 'unknown', 'RoleName' => '未知', 'UserCategoryID' => ''];
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
        preg_match_all('/TaskActivity\(actTeacherId(?:\.toString\(\)|.join\(\'\,\'\)),[^,]*,"(.+)","(.+)\(.*\)","(.+)","(.+)","(.+)",null,null,assistantName,"",""\);((?:\s*index =\d+\*unitCount\+\d+;\s*.*\s)+)/', $html, $rows, PREG_SET_ORDER);
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

    private function parseTeacherExams(string $html): array
    {
        $result = [];
        foreach (preg_split('/(?=<div id="toolbar[^"]*")/i', $html) as $section) {
            $title = preg_match('/bg\.ui\.toolbar\("[^"]+",\'([^\']*)\'/', $section, $match) ? $this->cleanHtmlCell($match[1]) : '';
            foreach ($this->tableRows($section) as $cells) {
                if (count($cells) < 7 || $cells[0] === '') {
                    continue;
                }
                $item = ['Category' => $title, 'CourseID' => $cells[0], 'CourseName' => $cells[1], 'Department' => $cells[2], 'Credit' => $cells[3], 'StudentCount' => '', 'Invigilators' => '', 'ExamTime' => '', 'ExamRoom' => ''];
                if (count($cells) >= 8) {
                    $item['Invigilators'] = $cells[4];
                    $item['StudentCount'] = $cells[5];
                    $item['ExamTime'] = $cells[6];
                    $item['ExamRoom'] = $cells[7];
                } else {
                    $item['StudentCount'] = $cells[4];
                    $item['ExamTime'] = $cells[5];
                    $item['ExamRoom'] = $cells[6];
                }
                $result[] = $item;
            }
        }
        return $result;
    }

    private function parseFreeRooms(string $html): array
    {
        $rooms = [];
        foreach ($this->tableRows($html) as $cells) {
            if (count($cells) >= 6 && $cells[0] !== '') {
                $rooms[] = ['Index' => $cells[0], 'Name' => $cells[1], 'Building' => $cells[2], 'Campus' => $cells[3], 'TypeName' => $cells[4], 'Capacity' => $cells[5]];
            }
        }
        return $rooms;
    }

    private function tableRows(string $html): array
    {
        preg_match_all('/<tr[^>]*>(.*?)<\/tr>/is', $html, $rows);
        $result = [];
        foreach ($rows[1] as $row) {
            preg_match_all('/<td[^>]*>(.*?)<\/td>/is', $row, $cells);
            if (count($cells[1]) > 0) {
                $result[] = array_map(fn (string $cell): string => $this->cleanHtmlCell($cell), $cells[1]);
            }
        }
        return $result;
    }

    private function cleanHtmlCell(string $value): string
    {
        return trim(preg_replace('/\s+/', ' ', html_entity_decode(strip_tags($value), ENT_QUOTES | ENT_HTML5, 'UTF-8')));
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

    private function escapeIcsText(string $value): string
    {
        return str_replace(["\\", "\n", "\r", ';', ','], ["\\\\", "\\n", '', "\\;", "\\,"], $value);
    }

    private function foldIcsLine(string $line): string
    {
        if (mb_strlen($line, 'UTF-8') <= 75) {
            return $line . "\r\n";
        }
        $parts = [];
        while (mb_strlen($line, 'UTF-8') > 75) {
            $parts[] = mb_substr($line, 0, 75, 'UTF-8');
            $line = mb_substr($line, 75, null, 'UTF-8');
        }
        $parts[] = $line;
        return implode("\r\n ", $parts) . "\r\n";
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

    private function authServerBase(string $loginUrl): string
    {
        $parts = parse_url($loginUrl);
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
