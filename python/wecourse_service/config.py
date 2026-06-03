from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path


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


def read_config() -> Config:
    config_path = Path(
        os.getenv("WECOURSE_CONFIG", Path(__file__).resolve().parents[2] / "config.json")
    ).resolve()
    with config_path.open("r", encoding="utf-8") as file:
        data = json.load(file)
    return Config(**data)
