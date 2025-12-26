#!/usr/bin/env python3
"""
QA Report Generator для ocserv-portal

Автоматически запускает:
- golangci-lint v2.7.2 внутри podman контейнера
- go test с coverage
- go vet, staticcheck (если доступны)
- Собирает метрики (errors, warnings, coverage %)
- Генерирует markdown отчет

Usage:
    python3 scripts/qa_report.py --container avndr-vpn-portal-backend --output docs/tmp/reports/
"""

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple


class Colors:
    """ANSI цвета для терминала"""
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    MAGENTA = '\033[0;35m'
    CYAN = '\033[0;36m'
    NC = '\033[0m'  # No Color


class QAReport:
    """Генератор QA отчетов"""

    def __init__(self, container_name: str, output_dir: str):
        self.container_name = container_name
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
        self.timestamp = datetime.now()
        self.metrics = {
            'golangci_lint': {'errors': 0, 'warnings': 0, 'issues': []},
            'go_test': {'passed': 0, 'failed': 0, 'skipped': 0, 'coverage': 0.0},
            'go_vet': {'issues': 0, 'errors': []},
            'staticcheck': {'issues': 0, 'errors': []},
        }

    def log_info(self, message: str):
        """Вывод информационного сообщения"""
        print(f"{Colors.BLUE}ℹ️  {message}{Colors.NC}")

    def log_success(self, message: str):
        """Вывод успешного сообщения"""
        print(f"{Colors.GREEN}✅ {message}{Colors.NC}")

    def log_warning(self, message: str):
        """Вывод предупреждения"""
        print(f"{Colors.YELLOW}⚠️  {message}{Colors.NC}")

    def log_error(self, message: str):
        """Вывод ошибки"""
        print(f"{Colors.RED}❌ {message}{Colors.NC}")

    def separator(self):
        """Вывод разделителя"""
        print("")
        print("═" * 70)
        print("")

    def run_command(self, cmd: List[str], capture_output: bool = True) -> Tuple[int, str, str]:
        """
        Выполнение команды и возврат результата

        Returns:
            (exit_code, stdout, stderr)
        """
        try:
            result = subprocess.run(
                cmd,
                capture_output=capture_output,
                text=True,
                timeout=600  # 10 минут максимум
            )
            return result.returncode, result.stdout, result.stderr
        except subprocess.TimeoutExpired:
            self.log_error(f"Command timeout: {' '.join(cmd)}")
            return 1, "", "Command timeout"
        except Exception as e:
            self.log_error(f"Command failed: {e}")
            return 1, "", str(e)

    def check_container_running(self) -> bool:
        """Проверка, что контейнер запущен"""
        self.log_info(f"Checking if container '{self.container_name}' is running...")
        exit_code, stdout, _ = self.run_command([
            "podman", "ps", "--format", "{{.Names}}"
        ])

        if exit_code != 0:
            self.log_error("Failed to list containers")
            return False

        running_containers = stdout.strip().split('\n')
        if self.container_name in running_containers:
            self.log_success(f"Container '{self.container_name}' is running")
            return True
        else:
            self.log_error(f"Container '{self.container_name}' is not running")
            self.log_info("Start it with: make compose-dev")
            return False

    def run_golangci_lint(self):
        """Запуск golangci-lint внутри контейнера"""
        self.separator()
        self.log_info("Running golangci-lint...")

        cmd = [
            "podman", "exec", self.container_name,
            "golangci-lint", "run", "--out-format", "json", "./..."
        ]

        exit_code, stdout, stderr = self.run_command(cmd)

        if exit_code == 0 and not stdout.strip():
            self.log_success("golangci-lint: No issues found!")
            self.metrics['golangci_lint']['errors'] = 0
            self.metrics['golangci_lint']['warnings'] = 0
            return

        # Парсинг JSON вывода
        try:
            data = json.loads(stdout) if stdout else {"Issues": []}
            issues = data.get("Issues", [])

            for issue in issues:
                severity = issue.get("Severity", "").lower()
                if severity == "error":
                    self.metrics['golangci_lint']['errors'] += 1
                else:
                    self.metrics['golangci_lint']['warnings'] += 1

                self.metrics['golangci_lint']['issues'].append({
                    'file': issue.get('Pos', {}).get('Filename', ''),
                    'line': issue.get('Pos', {}).get('Line', 0),
                    'linter': issue.get('FromLinter', ''),
                    'severity': severity,
                    'message': issue.get('Text', '')
                })

            self.log_warning(
                f"golangci-lint: {self.metrics['golangci_lint']['errors']} errors, "
                f"{self.metrics['golangci_lint']['warnings']} warnings"
            )

        except json.JSONDecodeError:
            self.log_error("Failed to parse golangci-lint JSON output")
            if stderr:
                self.log_error(f"stderr: {stderr}")

    def run_go_test(self):
        """Запуск go test с coverage"""
        self.separator()
        self.log_info("Running go test with coverage...")

        cmd = [
            "podman", "exec", self.container_name,
            "go", "test", "-v", "-short", "-race", "-coverprofile=/tmp/coverage.out",
            "-covermode=atomic", "./..."
        ]

        exit_code, stdout, stderr = self.run_command(cmd)

        # Парсинг вывода тестов
        passed = len(re.findall(r'--- PASS:', stdout))
        failed = len(re.findall(r'--- FAIL:', stdout))
        skipped = len(re.findall(r'--- SKIP:', stdout))

        self.metrics['go_test']['passed'] = passed
        self.metrics['go_test']['failed'] = failed
        self.metrics['go_test']['skipped'] = skipped

        # Получение coverage
        cmd_coverage = [
            "podman", "exec", self.container_name,
            "go", "tool", "cover", "-func=/tmp/coverage.out"
        ]

        exit_code, stdout, _ = self.run_command(cmd_coverage)

        if exit_code == 0:
            # Последняя строка содержит total coverage
            lines = stdout.strip().split('\n')
            if lines:
                last_line = lines[-1]
                match = re.search(r'(\d+\.\d+)%', last_line)
                if match:
                    self.metrics['go_test']['coverage'] = float(match.group(1))

        if failed > 0:
            self.log_error(f"Tests: {passed} passed, {failed} failed, {skipped} skipped")
        else:
            self.log_success(f"Tests: {passed} passed, {skipped} skipped")

        self.log_info(f"Coverage: {self.metrics['go_test']['coverage']:.2f}%")

    def run_go_vet(self):
        """Запуск go vet"""
        self.separator()
        self.log_info("Running go vet...")

        cmd = [
            "podman", "exec", self.container_name,
            "go", "vet", "./..."
        ]

        exit_code, stdout, stderr = self.run_command(cmd)

        if exit_code == 0:
            self.log_success("go vet: No issues found!")
        else:
            issues = stderr.strip().split('\n')
            self.metrics['go_vet']['issues'] = len([i for i in issues if i])
            self.metrics['go_vet']['errors'] = issues
            self.log_warning(f"go vet: {self.metrics['go_vet']['issues']} issues found")

    def run_staticcheck(self):
        """Запуск staticcheck (если установлен)"""
        self.separator()
        self.log_info("Running staticcheck...")

        cmd = [
            "podman", "exec", self.container_name,
            "staticcheck", "./..."
        ]

        exit_code, stdout, stderr = self.run_command(cmd)

        if exit_code == 127:  # Command not found
            self.log_warning("staticcheck not installed, skipping")
            return

        if exit_code == 0:
            self.log_success("staticcheck: No issues found!")
        else:
            issues = stdout.strip().split('\n')
            self.metrics['staticcheck']['issues'] = len([i for i in issues if i])
            self.metrics['staticcheck']['errors'] = issues
            self.log_warning(f"staticcheck: {self.metrics['staticcheck']['issues']} issues found")

    def generate_markdown_report(self) -> str:
        """Генерация markdown отчета"""
        date_str = self.timestamp.strftime("%Y-%m-%d")
        time_str = self.timestamp.strftime("%H:%M:%S")

        report = f"""# QA Report - ocserv-portal

![Status](https://img.shields.io/badge/Status-{'Pass' if self._is_passing() else 'Fail'}-{'green' if self._is_passing() else 'red'})
![Date](https://img.shields.io/badge/Date-{date_str}-blue)
![Coverage](https://img.shields.io/badge/Coverage-{self.metrics['go_test']['coverage']:.1f}%25-{'green' if self.metrics['go_test']['coverage'] >= 80 else 'yellow'})

---

## 📋 Метаданные

| Параметр | Значение |
|----------|----------|
| **Дата создания** | {date_str} {time_str} |
| **Контейнер** | `{self.container_name}` |
| **Общий статус** | {'✅ PASS' if self._is_passing() else '❌ FAIL'} |

---

## 📊 Сводка результатов

### golangci-lint

| Метрика | Значение |
|---------|----------|
| Errors | {self.metrics['golangci_lint']['errors']} |
| Warnings | {self.metrics['golangci_lint']['warnings']} |
| Total Issues | {len(self.metrics['golangci_lint']['issues'])} |

"""

        # Детали golangci-lint issues
        if self.metrics['golangci_lint']['issues']:
            report += "\n#### Top 10 golangci-lint Issues\n\n"
            report += "| File | Line | Linter | Severity | Message |\n"
            report += "|------|------|--------|----------|----------|\n"

            for issue in self.metrics['golangci_lint']['issues'][:10]:
                file_short = issue['file'].split('/')[-1] if issue['file'] else 'N/A'
                report += f"| {file_short} | {issue['line']} | {issue['linter']} | {issue['severity']} | {issue['message'][:50]}... |\n"

        report += f"""

### Go Tests

| Метрика | Значение |
|---------|----------|
| Passed | {self.metrics['go_test']['passed']} |
| Failed | {self.metrics['go_test']['failed']} |
| Skipped | {self.metrics['go_test']['skipped']} |
| **Coverage** | **{self.metrics['go_test']['coverage']:.2f}%** |

### go vet

| Метрика | Значение |
|---------|----------|
| Issues | {self.metrics['go_vet']['issues']} |

### staticcheck

| Метрика | Значение |
|---------|----------|
| Issues | {self.metrics['staticcheck']['issues']} |

---

## 🎯 Coverage Goals

| Компонент | Current | Target | Status |
|-----------|---------|--------|--------|
| Overall | {self.metrics['go_test']['coverage']:.1f}% | 80%+ | {'✅' if self.metrics['go_test']['coverage'] >= 80 else '⚠️'} |
| Auth Service | N/A | 95%+ | - |
| TOTP Service | N/A | 95%+ | - |
| Repositories | N/A | 85%+ | - |

---

## ✅ Quality Criteria

| Критерий | Статус |
|----------|--------|
| golangci-lint errors = 0 | {'✅' if self.metrics['golangci_lint']['errors'] == 0 else '❌'} |
| Tests passing | {'✅' if self.metrics['go_test']['failed'] == 0 else '❌'} |
| Coverage >= 80% | {'✅' if self.metrics['go_test']['coverage'] >= 80 else '❌'} |
| go vet clean | {'✅' if self.metrics['go_vet']['issues'] == 0 else '❌'} |

---

## 🔧 Рекомендации

"""

        # Генерация рекомендаций
        if self.metrics['golangci_lint']['errors'] > 0:
            report += f"- ⚠️ Исправить {self.metrics['golangci_lint']['errors']} критичных ошибок golangci-lint\n"

        if self.metrics['go_test']['failed'] > 0:
            report += f"- ❌ Исправить {self.metrics['go_test']['failed']} падающих тестов\n"

        if self.metrics['go_test']['coverage'] < 80:
            report += f"- 📈 Увеличить покрытие тестами до 80%+ (текущее: {self.metrics['go_test']['coverage']:.1f}%)\n"

        if self.metrics['go_vet']['issues'] > 0:
            report += f"- 🔍 Исправить {self.metrics['go_vet']['issues']} проблем go vet\n"

        if self._is_passing():
            report += "- ✅ Все проверки пройдены! Готово к коммиту.\n"

        report += f"""

---

## 📝 Команды для исправления

```bash
# Запуск линтера с автофиксом
podman exec {self.container_name} golangci-lint run --fix ./...

# Запуск тестов
podman exec {self.container_name} go test -v ./...

# Просмотр coverage в HTML
podman exec {self.container_name} go test -coverprofile=/tmp/coverage.out ./...
podman exec {self.container_name} go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html
```

---

> **Автоматически сгенерировано:** {self.timestamp.strftime("%Y-%m-%d %H:%M:%S")}
> **Скрипт:** `scripts/qa_report.py`
"""

        return report

    def _is_passing(self) -> bool:
        """Проверка, что все критерии качества выполнены"""
        return (
            self.metrics['golangci_lint']['errors'] == 0 and
            self.metrics['go_test']['failed'] == 0 and
            self.metrics['go_test']['coverage'] >= 80.0 and
            self.metrics['go_vet']['issues'] == 0
        )

    def save_report(self, report: str):
        """Сохранение отчета в файл"""
        date_str = self.timestamp.strftime("%Y-%m-%d")
        filename = f"{date_str}_qa-report.md"
        filepath = self.output_dir / filename

        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(report)

        self.log_success(f"Report saved to: {filepath}")
        return filepath

    def run(self):
        """Главная функция выполнения"""
        print("")
        print("╔" + "═" * 68 + "╗")
        print("║" + " " * 18 + "QA Report Generator" + " " * 31 + "║")
        print("║" + " " * 20 + "ocserv-portal" + " " * 35 + "║")
        print("╚" + "═" * 68 + "╝")
        print("")

        # Проверка контейнера
        if not self.check_container_running():
            return 1

        # Запуск всех проверок
        self.run_golangci_lint()
        self.run_go_test()
        self.run_go_vet()
        self.run_staticcheck()

        # Генерация отчета
        self.separator()
        self.log_info("Generating markdown report...")
        report = self.generate_markdown_report()
        filepath = self.save_report(report)

        # Финальная сводка
        self.separator()
        if self._is_passing():
            self.log_success("🎉 All quality checks passed!")
        else:
            self.log_warning("⚠️  Some quality checks failed. See report for details.")

        print("")
        print(f"📄 Report: {filepath}")
        print("")

        return 0 if self._is_passing() else 1


def main():
    """Точка входа"""
    parser = argparse.ArgumentParser(
        description="QA Report Generator для ocserv-portal",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )

    parser.add_argument(
        '--container',
        type=str,
        default='avndr-vpn-portal-backend',
        help='Имя Podman контейнера (default: avndr-vpn-portal-backend)'
    )

    parser.add_argument(
        '--output',
        type=str,
        default='docs/tmp/reports/',
        help='Директория для сохранения отчета (default: docs/tmp/reports/)'
    )

    args = parser.parse_args()

    qa_report = QAReport(container_name=args.container, output_dir=args.output)
    exit_code = qa_report.run()

    sys.exit(exit_code)


if __name__ == '__main__':
    main()
