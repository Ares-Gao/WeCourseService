from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class Config:
    SchoolName: str
    MangerType: str
    MangerURL: str
    CalendarFirst: str
    SocketPort: int
    LoginType: str = "direct"
    AuthServerURL: str = ""
    ServiceURL: str = ""
    AuthServerAutoCaptcha: bool = True
    AuthServerCaptchaRetries: int = 3
    CalendarTimezone: str = "Asia/Shanghai"
    CalendarName: str = "微课表"
    ClassTimeSlots: list[dict[str, str]] | None = None


def read_config() -> Config:
    config_path = Path(
        os.getenv("WECOURSE_CONFIG", Path(__file__).resolve().parents[2] / "config.json")
    ).resolve()
    with config_path.open("r", encoding="utf-8") as file:
        data: dict[str, Any] = json.load(file)
    return Config(**data)
