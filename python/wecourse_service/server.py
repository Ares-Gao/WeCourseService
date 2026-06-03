from __future__ import annotations

import asyncio
import json
from typing import Any

from websockets.server import serve

from .config import read_config
from .supwisdom import (
    get_account,
    get_course,
    get_day_course,
    get_grade,
    get_photo,
    get_semesters,
    get_teacher,
    get_user_login,
    get_week_course,
    get_week_course_new,
    get_week_time,
)

BUILD = "python-0.1.0"


def dispatch(message: str) -> str:
    payload: dict[str, Any] = json.loads(message)
    request_type = payload.get("Type", "")
    username = payload.get("UserName", "")
    password = payload.get("PassWord", "")
    week = int(payload.get("Week", 0))
    config = read_config()

    if request_type == "allcourse":
        return get_course(username, password)
    if request_type == "daycourse":
        return get_day_course(username, password)
    if request_type == "course":
        return get_week_course(username, password, week)
    if request_type == "weekcourse":
        return get_week_course_new(username, password, week)
    if request_type == "account":
        return get_account(username, password)
    if request_type == "login":
        return get_user_login(username, password)
    if request_type == "week":
        return get_week_time(config.CalendarFirst)
    if request_type == "teacher":
        return get_teacher(username, password)
    if request_type == "photo":
        return get_photo(username, password)
    if request_type == "grade":
        return get_grade(username, password)
    if request_type == "semester":
        return get_semesters(username, password)
    return json.dumps({"Type": request_type, "Data": "unsupported request type"}, ensure_ascii=False, indent="\t")


async def handler(websocket) -> None:
    async for message in websocket:
        try:
            await websocket.send(await asyncio.to_thread(dispatch, message))
        except Exception as exc:
            await websocket.send(json.dumps({"Type": "error", "Data": str(exc)}, ensure_ascii=False, indent="\t"))


async def run() -> None:
    config = read_config()
    print("Websocket服务开始运行")
    print("固件版本：" + BUILD)
    print("学校名称：" + config.SchoolName)
    print("绑定端口：" + str(config.SocketPort))
    async with serve(handler, "0.0.0.0", config.SocketPort):
        await asyncio.Future()


def main() -> None:
    asyncio.run(run())
