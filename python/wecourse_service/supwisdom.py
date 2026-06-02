from __future__ import annotations

import base64
import hashlib
import json
import re
from datetime import datetime
from time import sleep, time
from typing import Any

import requests

from .config import read_config


USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0"
LOGIN_MARKER = '<a href="/eams/security/my.action" target="_blank" title="查看详情" style="color:#ffffff">'

_course_cache: dict[str, tuple[float, str, list[dict[str, Any]]]] = {}


def _json_response(response_type: str, data: Any) -> str:
    return json.dumps({"Type": response_type, "Data": data}, ensure_ascii=False, indent="\t")


def _base_url() -> str:
    return read_config().MangerURL.rstrip("/") + "/"


def _extract_password_salt(login_html: str) -> str:
    marker = "CryptoJS.SHA1("
    index = login_html.find(marker)
    if index == -1:
        raise ValueError("登录页未找到密码盐值")
    return login_html[index + 15 : index + 52]


def _login(username: str, password: str) -> requests.Session:
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

    marker = 'bg.form.addInput(form,"ids","'
    start = page.text.find(marker)
    if start == -1:
        raise ValueError("未找到课表 ids")
    raw = page.text[start + len(marker) : start + len(marker) + 21]
    ids = raw[: raw.find('");')]

    response = session.post(
        base_url + "eams/courseTableForStd!courseTable.action",
        data={
            "ignoreHead": "1",
            "showPrintAndExport": "1",
            "setting.kind": "std",
            "startWeek": "",
            "semester.id": "30",
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
        r"TaskActivity\(actTeacherId.join\(','\),actTeacherName.join\(','\),\"(.*)\","
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
                "CourseName": match.group(2),
                "RoomID": match.group(3),
                "RoomName": match.group(4),
                "Weeks": match.group(5),
                "CourseTimes": course_times,
            }
        )
    return courses


def get_user_login(username: str, password: str) -> str:
    try:
        session = _login(username, password)
    except Exception:
        return _json_response("login", "登录失败")
    _logout(session)
    return _json_response("login", "登录成功")


def get_course(username: str, password: str) -> str:
    cache_item = _course_cache.get(username)
    if cache_item and time() - cache_item[0] < 3600:
        return cache_item[1]

    session = _login(username, password)
    try:
        html = _course_table_html(session)
        teachers = _parse_teachers(html)
        courses = _parse_courses(html)
        response = _json_response("allcourse", courses)
        _course_cache[username] = (time(), response, teachers)
        return response
    finally:
        _logout(session)


def get_teacher(username: str, password: str) -> str:
    session = _login(username, password)
    try:
        return _json_response("teacher", _parse_teachers(_course_table_html(session)))
    finally:
        _logout(session)


def _teacher_cache(username: str) -> list[dict[str, str]]:
    cache_item = _course_cache.get(username)
    return cache_item[2] if cache_item else []


def get_week_course(username: str, password: str, week: int, response_type: str = "course") -> str:
    result = json.loads(get_course(username, password))
    courses = result.get("Data", [])
    teachers = _teacher_cache(username)
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


def get_week_course_new(username: str, password: str, week: int) -> str:
    result = json.loads(get_course(username, password))
    courses = result.get("Data", [])
    teachers = _teacher_cache(username)
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


def get_day_course(username: str, password: str) -> str:
    week = int(json.loads(get_week_time(read_config().CalendarFirst))["Data"])
    weekday = datetime.now().weekday()
    result = json.loads(get_course(username, password))
    courses = result.get("Data", [])
    teachers = _teacher_cache(username)
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


def get_account(username: str, password: str) -> str:
    session = _login(username, password)
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


def get_photo(username: str, password: str) -> str:
    session = _login(username, password)
    try:
        response = session.get(_base_url() + f"eams/showSelfAvatar.action?user.name={username}", timeout=15)
        response.raise_for_status()
        data = base64.b64encode(response.content).decode("ascii")
        return _json_response("photo", "data:image/jpg;base64," + data)
    finally:
        _logout(session)


def get_grade(username: str, password: str) -> str:
    session = _login(username, password)
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
