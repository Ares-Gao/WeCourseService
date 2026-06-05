from __future__ import annotations

import base64
import hashlib
import json
import random
import re
from dataclasses import replace
from datetime import datetime, timedelta
from time import sleep, time
from typing import Any
from urllib.parse import urlparse

import requests
import urllib3
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad

from .config import read_config


USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0"
LOGIN_MARKER = '<a href="/eams/security/my.action" target="_blank" title="查看详情" style="color:#ffffff">'
AUTH_LOGIN_MARKER = "统一身份认证平台"
AES_CHARS = "ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678"
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

_course_cache: dict[str, tuple[float, str, list[dict[str, Any]]]] = {}
_ocr = None

DEFAULT_CLASS_TIME_SLOTS = [
    {"Start": "08:00", "End": "08:45"},
    {"Start": "08:55", "End": "09:40"},
    {"Start": "10:00", "End": "10:45"},
    {"Start": "10:55", "End": "11:40"},
    {"Start": "14:00", "End": "14:45"},
    {"Start": "14:55", "End": "15:40"},
    {"Start": "16:00", "End": "16:45"},
    {"Start": "16:55", "End": "17:40"},
    {"Start": "19:00", "End": "19:45"},
    {"Start": "19:55", "End": "20:40"},
    {"Start": "20:50", "End": "21:35"},
    {"Start": "21:45", "End": "22:30"},
]


def _json_response(response_type: str, data: Any) -> str:
    return json.dumps({"Type": response_type, "Data": data}, ensure_ascii=False, indent="\t")


def _base_url() -> str:
    return read_config().MangerURL.rstrip("/") + "/"


def _random_string(length: int) -> str:
    return "".join(random.choice(AES_CHARS) for _ in range(length))


def _authserver_encrypt_password(password: str, salt: str) -> str:
    key = salt.strip().encode("utf-8")
    iv = _random_string(16).encode("utf-8")
    payload = (_random_string(64) + password).encode("utf-8")
    cipher = AES.new(key, AES.MODE_CBC, iv)
    return base64.b64encode(cipher.encrypt(pad(payload, AES.block_size))).decode("ascii")


def _input_value(html: str, name: str = "", element_id: str = "") -> str:
    if element_id:
        match = re.search(rf"<input[^>]*id=[\"']{re.escape(element_id)}[\"'][^>]*>", html, re.I)
    else:
        match = re.search(rf"<input[^>]*name=[\"']{re.escape(name)}[\"'][^>]*>", html, re.I)
    if not match:
        return ""
    value = re.search(r"value=[\"']([^\"']*)", match.group(0), re.I)
    return value.group(1) if value else ""


def _authserver_base(login_url: str) -> str:
    parsed = urlparse(login_url)
    path = parsed.path
    marker = "/login"
    context_path = path[: path.find(marker)] if marker in path else "/authserver"
    return f"{parsed.scheme}://{parsed.netloc}{context_path}"


def _need_authserver_captcha(session: requests.Session, auth_base: str, username: str, html: str) -> bool:
    if re.search(r'var\s+_badCredentialsCount\s*=\s*"0"', html):
        return True
    try:
        response = session.get(
            auth_base + "/checkNeedCaptcha.htl",
            params={"username": username},
            timeout=15,
            verify=False,
        )
        response.raise_for_status()
        return bool(response.json().get("isNeed"))
    except Exception:
        return "captchaDiv" in html and "getCaptcha.htl" in html


def _recognize_authserver_captcha(session: requests.Session, auth_base: str) -> str:
    global _ocr
    if _ocr is None:
        import ddddocr

        _ocr = ddddocr.DdddOcr(show_ad=False)
    response = session.get(auth_base + "/getCaptcha.htl", params={str(int(time() * 1000)): ""}, timeout=15, verify=False)
    response.raise_for_status()
    return re.sub(r"[^0-9A-Za-z]", "", _ocr.classification(response.content)).strip()


def _extract_password_salt(login_html: str) -> str:
    marker = "CryptoJS.SHA1("
    index = login_html.find(marker)
    if index == -1:
        raise ValueError("登录页未找到密码盐值")
    return login_html[index + 15 : index + 52]


def _extract_course_table_params(html: str) -> tuple[str, str]:
    ids_match = re.search(r'bg\.form\.addInput\(form,\s*"ids",\s*"([^"]+)"', html)
    if not ids_match:
        raise ValueError("未找到课表 ids")

    semester_patterns = [
        r'name=["\']semester\.id["\'][^>]*value=["\']([^"\']+)["\']',
        r'semesterCalendar\(\{[^}]*value:\s*"([^"]+)"',
        r"semesterCalendar\(\{[^}]*value:\s*'([^']+)'",
        r'bg\.form\.addInput\(form,\s*"semester\.id",\s*"([^"]+)"',
    ]
    for pattern in semester_patterns:
        semester_match = re.search(pattern, html, re.I)
        if semester_match and semester_match.group(1):
            return ids_match.group(1), semester_match.group(1)

    raise ValueError("未找到 semester.id")


def _clean_js_text(text: str) -> str:
    return text.replace('"+periodInfo+"', "").replace("\\\"", "\"")


def _login(username: str, password: str, login_type: str = "", authserver_url: str = "") -> requests.Session:
    config = read_config()
    if login_type or authserver_url:
        config = replace(
            config,
            LoginType=login_type or config.LoginType,
            AuthServerURL=authserver_url or config.AuthServerURL,
        )
    if config.LoginType.lower() == "authserver":
        return _authserver_login(username, password, config)

    session = requests.Session()
    base_url = _base_url()

    login_page = session.get(base_url + "eams/login.action", timeout=15)
    login_page.raise_for_status()
    salt = _extract_password_salt(login_page.text)
    hashed_password = hashlib.sha1((salt + password).encode("utf-8")).hexdigest()

    sleep(1)
    response = session.post(
        base_url + "eams/login.action",
        data={
            "username": username,
            "password": hashed_password,
            "session_locale": "zh_CN",
        },
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "User-Agent": USER_AGENT,
        },
        timeout=15,
    )
    response.raise_for_status()
    if LOGIN_MARKER not in response.text:
        raise ValueError("登录失败")
    return session


def _authserver_login(username: str, password: str, config) -> requests.Session:
    login_url = config.AuthServerURL
    if not login_url:
        raise ValueError("authserver 登录需要配置 AuthServerURL")

    retries = max(1, int(config.AuthServerCaptchaRetries or 1))
    last_error = "authserver 登录失败"
    for _ in range(retries):
        session = requests.Session()
        session.trust_env = False
        login_page = session.get(login_url, timeout=20, verify=False)
        login_page.raise_for_status()
        html = login_page.text
        salt = _input_value(html, element_id="pwdEncryptSalt")
        execution = _input_value(html, name="execution")
        if not salt or not execution:
            raise ValueError("authserver 登录页缺少 pwdEncryptSalt 或 execution")

        auth_base = _authserver_base(login_page.url)
        captcha = ""
        if config.AuthServerAutoCaptcha and _need_authserver_captcha(session, auth_base, username, html):
            captcha = _recognize_authserver_captcha(session, auth_base)
            if not captcha:
                last_error = "验证码识别为空"
                continue

        response = session.post(
            login_page.url,
            data={
                "username": username,
                "password": _authserver_encrypt_password(password, salt),
                "captcha": captcha,
                "_eventId": _input_value(html, name="_eventId") or "submit",
                "cllt": "userNameLogin",
                "dllt": _input_value(html, name="dllt") or "generalLogin",
                "lt": _input_value(html, name="lt"),
                "execution": execution,
                "rmShown": "1",
            },
            headers={
                "Origin": re.match(r"^https?://[^/]+", login_page.url).group(0),
                "Referer": login_page.url,
                "User-Agent": USER_AGENT,
            },
            timeout=30,
            verify=False,
            allow_redirects=True,
        )
        if response.status_code >= 400 and "authserver/login" in response.url:
            response.raise_for_status()
        if "authserver/login" not in response.url and "认证失败" not in response.text:
            return session

        last_error = "authserver 认证失败"
        if captcha and ("验证码" in response.text or "captcha" in response.text.lower()):
            session.close()
            continue
        session.close()

    raise ValueError(last_error)


def _logout(session: requests.Session) -> None:
    try:
        session.get(_base_url() + "eams/logout.action", timeout=10)
    finally:
        session.close()


def _course_table_html(session: requests.Session) -> str:
    base_url = _base_url()
    sleep(1)
    page = session.get(base_url + "eams/courseTableForStd.action", timeout=15)
    page.raise_for_status()

    ids, semester_id = _extract_course_table_params(page.text)

    response = session.post(
        base_url + "eams/courseTableForStd!courseTable.action",
        data={
            "ignoreHead": "1",
            "showPrintAndExport": "1",
            "setting.kind": "std",
            "startWeek": "",
            "semester.id": semester_id,
            "ids": ids,
        },
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "User-Agent": USER_AGENT,
        },
        timeout=20,
    )
    response.raise_for_status()
    if "课表格式说明" not in response.text:
        raise ValueError("课表获取失败")
    return response.text


def _semester_payload(session: requests.Session) -> list[dict[str, Any]]:
    page = session.get(_base_url() + "eams/courseTableForStd.action", timeout=15)
    page.raise_for_status()
    ids, semester_id = _extract_course_table_params(page.text)
    return [{"SemesterID": semester_id, "Ids": ids, "Current": True}]


def _parse_teachers(html: str) -> list[dict[str, str]]:
    row_pattern = re.compile(
        r"(?i)<td>(\d)</td>\s*<td>([:alpha:].+)</td>\s*<td>(.+)</td>\s*"
        r"<td>((\d)|(\d\.\d))</td>\s*<td>\s*<a href=.*\s.*\s.*\s.*>.*</a>\s*</td>\s*<td>(.*)</td>"
    )
    td_pattern = re.compile(r"(?i)<td>([^>]*)</td>")
    link_pattern = re.compile(r"(?i)>([^>]*)</a>")

    teachers: list[dict[str, str]] = []
    for row in row_pattern.finditer(html):
        tds = td_pattern.findall(row.group(0))
        links = link_pattern.findall(row.group(0))
        if len(tds) >= 5 and links:
            teachers.append(
                {
                    "CourseID": links[0],
                    "CourseName": tds[2],
                    "CourseCredit": tds[3],
                    "CourseTeacher": tds[4],
                }
            )
    return teachers


def _parse_courses(html: str) -> list[dict[str, Any]]:
    activity_pattern = re.compile(
        r"TaskActivity\(actTeacherId(?:\.toString\(\)|.join\(','\)),[^,]*,\"(.*)\","
        r'"(.*)\(.*\)","(.*)","(.*)","(.*)",null,null,assistantName,"",""\);'
        r"((?:\s*index =\d+\*unitCount\+\d+;\s*.*\s)+)"
    )
    index_pattern = re.compile(r"\s*index =(\d+)\*unitCount\+(\d+);\s*")

    courses: list[dict[str, Any]] = []
    for match in activity_pattern.finditer(html):
        course_times = []
        for index_text in match.group(6).split("table0.activities[index][table0.activities[index].length]=activity;"):
            index_match = index_pattern.search(index_text)
            if index_match:
                course_times.append(
                    {
                        "DayOfTheWeek": int(index_match.group(1)),
                        "TimeOfTheDay": int(index_match.group(2)),
                    }
                )
        courses.append(
            {
                "CourseID": match.group(1),
                "CourseName": _clean_js_text(match.group(2)),
                "RoomID": match.group(3),
                "RoomName": match.group(4),
                "Weeks": match.group(5),
                "CourseTimes": course_times,
            }
        )
    return courses


def get_user_login(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    try:
        session = _login(username, password, login_type, authserver_url)
    except Exception:
        return _json_response("login", "登录失败")
    _logout(session)
    return _json_response("login", "登录成功")


def _parse_home_ext_identity(html: str) -> dict[str, str]:
    identity = {"Role": "unknown", "RoleName": "未知", "UserCategoryID": ""}
    match = re.search(r"""<input[^>]+name=["']security\.userCategoryId["'][^>]*value=["']([^"']+)["']""", html, re.I | re.S)
    if match:
        identity["UserCategoryID"] = match.group(1).strip()
        if identity["UserCategoryID"] == "1":
            identity.update({"Role": "student", "RoleName": "学生"})
            return identity
        if identity["UserCategoryID"] == "2":
            identity.update({"Role": "teacher", "RoleName": "教师"})
            return identity

    if "courseTableForStd.action" in html or "stdDetail.action" in html or "学生" in html:
        identity.update({"Role": "student", "RoleName": "学生"})
        return identity
    if "courseTableForTeacher.action" in html or "teacherExamTable.action" in html or "教师" in html:
        identity.update({"Role": "teacher", "RoleName": "教师"})
        return identity
    return identity


def get_identity(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        page = session.get(_base_url() + "eams/homeExt.action", timeout=15)
        page.raise_for_status()
        return _json_response("identity", _parse_home_ext_identity(page.text))
    finally:
        _logout(session)


def _teacher_course_table_html(session: requests.Session) -> str:
    base_url = _base_url()
    page = session.get(base_url + "eams/courseTableForTeacher.action", timeout=15)
    page.raise_for_status()
    ids = re.search(r"""name=["']ids["'][^>]*value=["']([^"']+)["']""", page.text)
    semester = re.search(r"""semesterCalendar\(\{[^}]*value:["']([^"']+)["']""", page.text)
    if not ids or not semester:
        raise ValueError("teacher course table params not found")
    response = session.post(
        base_url + "eams/courseTableForTeacher!courseTable.action",
        data={
            "ignoreHead": "1",
            "setting.forSemester": "1",
            "ids": ids.group(1),
            "setting.kind": "teacher",
            "semester.id": semester.group(1),
        },
        timeout=15,
    )
    response.raise_for_status()
    return response.text


def get_teacher_course(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        return _json_response("teachercourse", _parse_courses(_teacher_course_table_html(session)))
    finally:
        _logout(session)


def _clean_html_cell(value: str) -> str:
    value = re.sub(r"(?is)<[^>]+>", "", value).replace("&nbsp;", " ")
    return " ".join(value.split())


def _table_rows(html: str) -> list[list[str]]:
    rows: list[list[str]] = []
    for row in re.finditer(r"(?is)<tr[^>]*>(.*?)</tr>", html):
        cells = [_clean_html_cell(cell.group(1)) for cell in re.finditer(r"(?is)<td[^>]*>(.*?)</td>", row.group(1))]
        if cells:
            rows.append(cells)
    return rows


def _parse_teacher_exams(html: str) -> list[dict[str, str]]:
    exams: list[dict[str, str]] = []
    for section in re.split(r'(?=<div id="toolbar[^"]*")', html):
        title = ""
        title_match = re.search(r"""bg\.ui\.toolbar\("[^"]+",'([^']*)'""", section)
        if title_match:
            title = _clean_html_cell(title_match.group(1))
        for cells in _table_rows(section):
            if len(cells) < 7 or not cells[0]:
                continue
            item = {
                "Category": title,
                "CourseID": cells[0],
                "CourseName": cells[1],
                "Department": cells[2],
                "Credit": cells[3],
                "StudentCount": "",
                "Invigilators": "",
                "ExamTime": "",
                "ExamRoom": "",
            }
            if len(cells) >= 8:
                item.update({"Invigilators": cells[4], "StudentCount": cells[5], "ExamTime": cells[6], "ExamRoom": cells[7]})
            else:
                item.update({"StudentCount": cells[4], "ExamTime": cells[5], "ExamRoom": cells[6]})
            exams.append(item)
    return exams


def get_teacher_exam(
    username: str,
    password: str,
    login_type: str = "",
    authserver_url: str = "",
    exam_batch_id: str = "",
) -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        base_url = _base_url()
        page = session.get(base_url + "eams/teacherExamTable.action", timeout=15)
        page.raise_for_status()
        if not exam_batch_id:
            for batch in _parse_teacher_exam_batches(page.text):
                if batch["Selected"]:
                    exam_batch_id = batch["ExamBatchID"]
                    break
        if not exam_batch_id:
            return _json_response("teacherexam", [])
        response = session.get(base_url + "eams/teacherExamTable!examAtivities.action?examBatch.id=" + exam_batch_id, timeout=15)
        response.raise_for_status()
        return _json_response("teacherexam", _parse_teacher_exams(response.text))
    finally:
        _logout(session)


def _parse_teacher_exam_batches(html: str) -> list[dict[str, Any]]:
    batches = []
    for match in re.finditer(r"""(?is)<option\s+value=["']([^"']+)["']([^>]*)>(.*?)</option>""", html):
        batches.append(
            {
                "ExamBatchID": match.group(1),
                "Name": _clean_html_cell(match.group(3)),
                "Selected": "selected" in match.group(2).lower(),
            }
        )
    return batches


def get_teacher_exam_batches(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        page = session.get(_base_url() + "eams/teacherExamTable.action", timeout=15)
        page.raise_for_status()
        return _json_response("teacherexambatch", _parse_teacher_exam_batches(page.text))
    finally:
        _logout(session)


def _parse_free_rooms(html: str) -> list[dict[str, str]]:
    rooms = []
    for cells in _table_rows(html):
        if len(cells) < 6 or not cells[0]:
            continue
        rooms.append(
            {
                "Index": cells[0],
                "Name": cells[1],
                "Building": cells[2],
                "Campus": cells[3],
                "TypeName": cells[4],
                "Capacity": cells[5],
            }
        )
    return rooms


def get_free_room(payload: dict[str, Any]) -> str:
    date_begin = payload.get("DateBegin") or datetime.now().strftime("%Y-%m-%d")
    time_begin = str(payload.get("TimeBegin", "1"))
    data = {
        "classroom.type.id": payload.get("ClassroomType", ""),
        "classroom.campus.id": payload.get("CampusID", ""),
        "classroom.building.id": payload.get("BuildingID", ""),
        "seats": payload.get("Seats", ""),
        "classroom.name": payload.get("ClassroomName", ""),
        "cycleTime.cycleCount": payload.get("CycleCount", "1"),
        "cycleTime.cycleType": payload.get("CycleType", "1"),
        "cycleTime.dateBegin": date_begin,
        "cycleTime.dateEnd": payload.get("DateEnd") or date_begin,
        "roomApplyTimeType": payload.get("RoomTimeType", "0"),
        "timeBegin": time_begin,
        "timeEnd": str(payload.get("TimeEnd", time_begin)),
    }
    response = requests.post(_base_url() + "eams/publicFree!search.action", data=data, timeout=15)
    response.raise_for_status()
    return _json_response("freeroom", _parse_free_rooms(response.text))


def get_course(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    cache_key = f"{login_type or read_config().LoginType}:{authserver_url}:{username}"
    cache_item = _course_cache.get(cache_key)
    if cache_item and time() - cache_item[0] < 3600:
        return cache_item[1]

    session = _login(username, password, login_type, authserver_url)
    try:
        html = _course_table_html(session)
        teachers = _parse_teachers(html)
        courses = _parse_courses(html)
        response = _json_response("allcourse", courses)
        _course_cache[cache_key] = (time(), response, teachers)
        return response
    finally:
        _logout(session)


def get_teacher(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        return _json_response("teacher", _parse_teachers(_course_table_html(session)))
    finally:
        _logout(session)


def get_semesters(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        return _json_response("semester", _semester_payload(session))
    finally:
        _logout(session)


def _teacher_cache(username: str, login_type: str = "", authserver_url: str = "") -> list[dict[str, str]]:
    cache_key = f"{login_type or read_config().LoginType}:{authserver_url}:{username}"
    cache_item = _course_cache.get(cache_key)
    return cache_item[2] if cache_item else []


def get_week_course(
    username: str,
    password: str,
    week: int,
    response_type: str = "course",
    login_type: str = "",
    authserver_url: str = "",
) -> str:
    result = json.loads(get_course(username, password, login_type, authserver_url))
    courses = result.get("Data", [])
    teachers = _teacher_cache(username, login_type, authserver_url)
    week_courses = []

    for course in courses:
        weeks = course.get("Weeks", "")
        if week >= len(weeks) or weeks[week] != "1":
            continue
        for teacher in teachers:
            if teacher["CourseID"] in course.get("CourseID", ""):
                times = ",".join(str(item["TimeOfTheDay"] + 1) for item in course.get("CourseTimes", []))
                week_courses.append(
                    {
                        "CourseName": teacher["CourseName"],
                        "TeacherName": teacher["CourseTeacher"],
                        "RoomName": course["RoomName"],
                        "DayOfTheWeek": course["CourseTimes"][0]["DayOfTheWeek"] if course.get("CourseTimes") else 0,
                        "TimeOfTheDay": times,
                    }
                )
    return _json_response(response_type, week_courses)


def get_week_course_new(username: str, password: str, week: int, login_type: str = "", authserver_url: str = "") -> str:
    result = json.loads(get_course(username, password, login_type, authserver_url))
    courses = result.get("Data", [])
    teachers = _teacher_cache(username, login_type, authserver_url)
    week_courses = []

    for course in courses:
        weeks = course.get("Weeks", "")
        if week >= len(weeks) or weeks[week] != "1":
            continue
        for teacher in teachers:
            if teacher["CourseID"] in course.get("CourseID", ""):
                week_courses.append(
                    {
                        "CourseName": teacher["CourseName"],
                        "TeacherName": teacher["CourseTeacher"],
                        "RoomName": course["RoomName"],
                        "CourseTimes": course.get("CourseTimes", []),
                    }
                )
    return _json_response("course", week_courses)


def _escape_ical_text(value: str) -> str:
    return value.replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "").replace(";", "\\;").replace(",", "\\,")


def _fold_ical_line(line: str) -> str:
    if len(line) <= 75:
        return line + "\r\n"
    return "\r\n ".join(line[i : i + 75] for i in range(0, len(line), 75)) + "\r\n"


def _course_ics(courses: list[dict[str, Any]]) -> str:
    config = read_config()
    slots = config.ClassTimeSlots or DEFAULT_CLASS_TIME_SLOTS
    first_monday = datetime.strptime(config.CalendarFirst, "%Y-%m-%d")
    timezone = config.CalendarTimezone or "Asia/Shanghai"
    calendar_name = config.CalendarName or f"{config.SchoolName}课表"
    now = datetime.utcnow().strftime("%Y%m%dT%H%M%SZ")
    lines = [
        "BEGIN:VCALENDAR\r\n",
        "VERSION:2.0\r\n",
        "PRODID:-//Ares-Gao//WeCourseService//CN\r\n",
        "CALSCALE:GREGORIAN\r\n",
        "METHOD:PUBLISH\r\n",
        _fold_ical_line("X-WR-CALNAME:" + _escape_ical_text(calendar_name)),
        _fold_ical_line("X-WR-TIMEZONE:" + timezone),
    ]
    for course in courses:
        day_times: dict[int, list[int]] = {}
        for item in course.get("CourseTimes", []):
            day_times.setdefault(int(item["DayOfTheWeek"]), []).append(int(item["TimeOfTheDay"]))
        for day, times in day_times.items():
            times.sort()
            start_slot, end_slot = times[0], times[-1]
            if start_slot >= len(slots) or end_slot >= len(slots):
                continue
            for week_index, enabled in enumerate(course.get("Weeks", "")):
                if week_index == 0 or enabled != "1":
                    continue
                date = first_monday + timedelta(days=(week_index - 1) * 7 + day)
                start_at = datetime.strptime(date.strftime("%Y-%m-%d") + " " + slots[start_slot]["Start"], "%Y-%m-%d %H:%M")
                end_at = datetime.strptime(date.strftime("%Y-%m-%d") + " " + slots[end_slot]["End"], "%Y-%m-%d %H:%M")
                uid_raw = f'{course.get("CourseID", "")}-{week_index}-{day}-{start_slot}'
                uid = re.sub(r"[^0-9A-Za-z_-]", "-", uid_raw) + "@wecourse.service"
                lines.extend(
                    [
                        "BEGIN:VEVENT\r\n",
                        _fold_ical_line("UID:" + uid),
                        "DTSTAMP:" + now + "\r\n",
                        _fold_ical_line("DTSTART;TZID=" + timezone + ":" + start_at.strftime("%Y%m%dT%H%M%S")),
                        _fold_ical_line("DTEND;TZID=" + timezone + ":" + end_at.strftime("%Y%m%dT%H%M%S")),
                        _fold_ical_line("SUMMARY:" + _escape_ical_text(course.get("CourseName", ""))),
                        _fold_ical_line("LOCATION:" + _escape_ical_text(course.get("RoomName", ""))),
                        _fold_ical_line(
                            "DESCRIPTION:"
                            + _escape_ical_text(f'CourseID: {course.get("CourseID", "")}\nRoomID: {course.get("RoomID", "")}')
                        ),
                        "END:VEVENT\r\n",
                    ]
                )
    lines.append("END:VCALENDAR\r\n")
    return "".join(lines)


def get_ics(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    result = json.loads(get_course(username, password, login_type, authserver_url))
    return _json_response("ics", _course_ics(result.get("Data", [])))


def get_day_course(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    week = int(json.loads(get_week_time(read_config().CalendarFirst))["Data"])
    weekday = datetime.now().weekday()
    result = json.loads(get_course(username, password, login_type, authserver_url))
    courses = result.get("Data", [])
    teachers = _teacher_cache(username, login_type, authserver_url)
    day_courses = []

    for course in courses:
        weeks = course.get("Weeks", "")
        course_times = course.get("CourseTimes", [])
        if week >= len(weeks) or weeks[week] != "1" or not course_times:
            continue
        if course_times[0]["DayOfTheWeek"] != weekday:
            continue
        for teacher in teachers:
            if teacher["CourseID"] in course.get("CourseID", ""):
                time_of_day = ",".join(str(item["TimeOfTheDay"] + 1) for item in course_times[:2])
                day_courses.append(
                    {
                        "CourseName": teacher["CourseName"],
                        "TeacherName": teacher["CourseTeacher"],
                        "TimeOfTheDay": time_of_day,
                        "SchoolWeek": str(week),
                    }
                )
    return _json_response("daycourse", day_courses)


def get_week_time(start_time: str) -> str:
    start = datetime.strptime(start_time + " 00:00:00", "%Y-%m-%d %H:%M:%S")
    now = datetime.now()
    week = round((now - start).total_seconds() / 60 / 60 / 24 / 7) + 1
    return _json_response("week", str(week))


def get_account(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        response = session.get(_base_url() + "eams/stdDetail.action", timeout=15)
        response.raise_for_status()
        stuinfo = re.findall(r"(?i)<td>([^>]*)</td>", response.text)
        student = {
            "FullName": stuinfo[0],
            "EnglishName": stuinfo[1],
            "Sex": stuinfo[2],
            "StartTime": stuinfo[11],
            "EndTime": stuinfo[12],
            "SchoolYear": stuinfo[4],
            "Type": f"{stuinfo[5]}({stuinfo[14]})",
            "System": stuinfo[8],
            "Specialty": stuinfo[9],
            "Class": stuinfo[18],
        }
        return _json_response("account", student)
    finally:
        _logout(session)


def get_photo(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        response = session.get(_base_url() + f"eams/showSelfAvatar.action?user.name={username}", timeout=15)
        response.raise_for_status()
        data = base64.b64encode(response.content).decode("ascii")
        return _json_response("photo", "data:image/jpg;base64," + data)
    finally:
        _logout(session)


def get_grade(username: str, password: str, login_type: str = "", authserver_url: str = "") -> str:
    session = _login(username, password, login_type, authserver_url)
    try:
        response = session.post(
            _base_url() + "eams/teach/grade/course/person!historyCourseGrade.action?projectType=MAJOR",
            headers={
                "Content-Type": "application/x-www-form-urlencoded",
                "User-Agent": USER_AGENT,
            },
            timeout=20,
        )
        response.raise_for_status()
        rows = re.findall(r"(?i)<tr>[\s\S]*?</tr>", response.text)[2:]
        grades = []
        for row in rows:
            tds = re.findall(r"(?i)<td.*>([^>]*)</td>", row)
            if len(tds) < 6:
                continue
            sup = re.findall(r"(?i)<sup.*>([^>]*)</sup>", row)
            grades.append(
                {
                    "CourseID": tds[1].strip("\n"),
                    "CourseName": sup[0] if sup else tds[3].strip("\t\r\n"),
                    "CourseTerm": tds[0].strip("\n"),
                    "CourseCredit": tds[4].strip("\n"),
                    "CourseGrade": tds[-2].strip("\t\n"),
                    "GradePoint": tds[-1].strip("\t\n"),
                }
            )
        return _json_response("grade", grades)
    finally:
        _logout(session)
