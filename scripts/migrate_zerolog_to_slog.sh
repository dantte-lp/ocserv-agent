#!/bin/bash
# Скрипт автоматической миграции zerolog → slog

set -e

PROJECT_ROOT="/opt/project/repositories/ocserv-agent"
cd "$PROJECT_ROOT"

echo "=== Миграция zerolog → slog ==="

# Функция замены в файле
migrate_file() {
    local file="$1"
    echo "Обработка: $file"

    # Замена импортов
    sed -i 's|"github.com/rs/zerolog"|"log/slog"|g' "$file"
    sed -i 's|"github.com/rs/zerolog/log"||g' "$file"

    # Замена типов
    sed -i 's|zerolog\.Logger|*slog.Logger|g' "$file"
    sed -i 's|zerolog\.New|slog.New|g' "$file"
    sed -i 's|zerolog\.Nop()|slog.New(slog.NewTextHandler(io.Discard, nil))|g' "$file"
    sed -i 's|zerolog\.NewTestWriter(t)|io.Discard|g' "$file"

    # Замена вызовов логирования - многострочные с .Str().Msg()
    # Эти паттерны обрабатываются вручную, т.к. sed плохо работает с многострочными заменами
}

# Обработка основных файлов
for file in internal/ocserv/*.go; do
    if [[ -f "$file" && ! "$file" =~ _test\.go$ ]]; then
        migrate_file "$file"
    fi
done

echo "=== Базовая миграция завершена ==="
echo "Требуется ручная доработка вызовов логирования"
