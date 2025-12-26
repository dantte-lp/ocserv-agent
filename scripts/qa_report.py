#!/usr/bin/env python3
"""
QA Report Generator для ocserv-agent

Автоматически запускает внутри podman контейнера:
- go build (компиляция)
- go test с coverage (тесты)
- golangci-lint (линтинг)
- staticcheck (статический анализ)
- gosec (сканирование безопасности)
- govulncheck (проверка уязвимостей)
- go vet (базовая проверка)
- gofmt (проверка форматирования)
- Генерирует markdown отчет

Usage:
    python3 scripts/qa_report.py --container ocserv-agent-qa
    python3 scripts/qa_report.py --fix --verbose
"""

import argparse
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any
