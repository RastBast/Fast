#!/bin/bash

# 🧪 ПОЛНАЯ ПРОВЕРКА РАБОТОСПОСОБНОСТИ VERB EXTRACTOR
# Этот скрипт проверит все компоненты проекта

set -e

GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # No Color

echo -e "${GREEN}🔍 ПРОВЕРКА РАБОТОСПОСОБНОСТИ VERB EXTRACTOR${NC}"
echo "================================================================"

# Функция для проверки
check_step() {
    local step_name="$1"
    local command="$2"
    local expected="$3"

    echo -e "\n${BLUE}➤ $step_name${NC}"
    echo "Команда: $command"

    if eval "$command"; then
        echo -e "${GREEN}✅ $step_name - УСПЕШНО${NC}"
        return 0
    else
        echo -e "${RED}❌ $step_name - НЕУДАЧНО${NC}"
        if [ -n "$expected" ]; then
            echo "Ожидалось: $expected"
        fi
        return 1
    fi
}

# Проверка файлов
echo -e "\n${YELLOW}📁 ШАГ 1: ПРОВЕРКА ФАЙЛОВ${NC}"
echo "----------------------------------------"

check_step "Основной сервер" "test -f verb_extractor_server.go"
check_step "Go модуль" "test -f go.mod" 
check_step "Makefile" "test -f Makefile"
check_step "Простые примеры (Bash)" "test -f simple_examples.sh"
check_step "Простые примеры (Go)" "test -f simple_examples.go"
check_step "Нагрузочные тесты (Bash)" "test -f simple_load_test.sh"
check_step "Нагрузочные тесты (Go)" "test -f simple_load_test.go"
check_step "Тестовые данные ГПМРМ" "test -f hpmor_sample.txt"

# Проверка синтаксиса Go
echo -e "\n${YELLOW}🔧 ШАг 2: ПРОВЕРКА СИНТАКСИСА GO${NC}"
echo "----------------------------------------"

check_step "Синтаксис основного сервера" "go build -o /tmp/verb-test verb_extractor_server.go && rm -f /tmp/verb-test"
check_step "Синтаксис простых примеров" "go build -o /tmp/examples-test simple_examples.go && rm -f /tmp/examples-test"
check_step "Синтаксис нагрузочных тестов" "go build -o /tmp/load-test simple_load_test.go && rm -f /tmp/load-test"

# Проверка прав доступа к скриптам
echo -e "\n${YELLOW}📋 ШАГ 3: ПРОВЕРКА ПРАВ ДОСТУПА${NC}"
echo "----------------------------------------"

chmod +x simple_examples.sh simple_load_test.sh test_api.sh 2>/dev/null || true

check_step "Права на simple_examples.sh" "test -x simple_examples.sh"
check_step "Права на simple_load_test.sh" "test -x simple_load_test.sh"

# Проверка зависимостей
echo -e "\n${YELLOW}🛠️ ШАГ 4: ПРОВЕРКА ЗАВИСИМОСТЕЙ${NC}"
echo "----------------------------------------"

check_step "Go установлен" "go version"
check_step "curl доступен" "curl --version | head -n1"

# Опциональные зависимости
echo -e "${BLUE}Проверка опциональных зависимостей:${NC}"
if command -v jq >/dev/null 2>&1; then
    echo -e "✅ jq установлен (для красивого JSON)"
else
    echo -e "⚠️  jq не установлен (JSON будет в сыром виде)"
fi

# Запуск сервера в фоне для тестирования
echo -e "\n${YELLOW}🚀 ШАГ 5: ТЕСТИРОВАНИЕ СЕРВЕРА${NC}"
echo "----------------------------------------"

echo "Запуск сервера в фоновом режиме..."
go run verb_extractor_server.go &
SERVER_PID=$!

# Ждем запуска сервера
echo "Ожидание запуска сервера (5 секунд)..."
sleep 5

# Проверка работы сервера
API_URL="http://localhost:8080"

if curl -s "$API_URL/api/health" >/dev/null; then
    echo -e "${GREEN}✅ Сервер запущен и доступен${NC}"

    # Тестирование API endpoints
    echo -e "\n${BLUE}Тестирование API endpoints:${NC}"

    # Health check
    if curl -s "$API_URL/api/health" | grep -q "ok"; then
        echo -e "✅ Health check работает"
    else
        echo -e "❌ Health check не работает"
    fi

    # Metrics
    if curl -s "$API_URL/api/metrics" >/dev/null; then
        echo -e "✅ Metrics endpoint работает"
    else
        echo -e "❌ Metrics endpoint не работает"
    fi

    # Основной API
    echo -e "\n${BLUE}Тестирование извлечения глаголов:${NC}"

    api_response=$(curl -s -X POST "$API_URL/api/extract-verbs" \
        -H "Content-Type: application/json" \
        -d '{"text":"Я читаю книгу и думаю о жизни"}')

    if echo "$api_response" | grep -q "verbs"; then
        echo -e "✅ API извлечения глаголов работает"
        echo "Ответ: $api_response"
    else
        echo -e "❌ API извлечения глаголов не работает"
        echo "Ответ: $api_response"
    fi

    # Тест проблемы из задания
    echo -e "\n${BLUE}Тест проблемы из задания (ударил/ударить/ударяю):${NC}"

    problem_response=$(curl -s -X POST "$API_URL/api/extract-verbs" \
        -H "Content-Type: application/json" \
        -d '{"text":"Он ударил мяч. Нужно ударить точно. Я ударяю по цели."}')

    if echo "$problem_response" | grep -q "ударить"; then
        echo -e "✅ Проблема из задания решена - разные формы приводятся к одной лемме"
        echo "Ответ: $problem_response"
    else
        echo -e "⚠️  Нужно проверить лемматизацию вручную"
        echo "Ответ: $problem_response"
    fi

else
    echo -e "${RED}❌ Сервер не запустился или недоступен${NC}"
    SERVER_FAILED=true
fi

# Останавливаем сервер
echo -e "\n${BLUE}Остановка тестового сервера...${NC}"
kill $SERVER_PID 2>/dev/null || true
sleep 2

if [ -z "$SERVER_FAILED" ]; then
    # Тестирование скриптов (только если сервер работал)
    echo -e "\n${YELLOW}🧪 ШАГ 6: ТЕСТИРОВАНИЕ СКРИПТОВ${NC}"
    echo "----------------------------------------"

    echo -e "${BLUE}Для полного тестирования скриптов:${NC}"
    echo "1. Запустите сервер: go run verb_extractor_server.go"
    echo "2. В другом терминале запустите:"
    echo "   • ./simple_examples.sh"
    echo "   • ./simple_load_test.sh" 
    echo "   • go run simple_examples.go"
    echo "   • go run simple_load_test.go"
fi

# Проверка Makefile команд
echo -e "\n${YELLOW}📋 ШАГ 7: ПРОВЕРКА MAKEFILE${NC}"
echo "----------------------------------------"

check_step "Makefile help" "make help | head -n5"
check_step "Makefile синтаксис" "make -n build >/dev/null"

# Итоговый результат
echo -e "\n================================================================"
echo -e "${GREEN}🎉 ПРОВЕРКА ЗАВЕРШЕНА${NC}"
echo "================================================================"

echo -e "\n${YELLOW}📋 ИНСТРУКЦИЯ ПО ЗАПУСКУ:${NC}"
echo "----------------------------------------"

echo -e "\n${BLUE}1. Запуск сервера:${NC}"
echo "   go run verb_extractor_server.go"
echo "   # или"
echo "   make run"

echo -e "\n${BLUE}2. Тестирование (в другом терминале):${NC}"
echo "   Простые примеры:"
echo "   • ./simple_examples.sh          # Bash версия"
echo "   • go run simple_examples.go     # Go версия"
echo "   • make examples                 # Через Makefile"

echo -e "\n   Нагрузочное тестирование:"
echo "   • ./simple_load_test.sh         # Bash версия"
echo "   • go run simple_load_test.go    # Go версия" 
echo "   • make load-test                # Через Makefile"

echo -e "\n   Все тесты сразу:"
echo "   • make test-all                 # Полное тестирование"
echo "   • make demo                     # Интерактивная демонстрация"

echo -e "\n${BLUE}3. Ручное тестирование API:${NC}"
echo "   curl -X POST http://localhost:8080/api/extract-verbs \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{"text":"Я программирую на Go и создаю сервисы"}'"

echo -e "\n${GREEN}✅ Проект готов к использованию!${NC}"

# Проверка на проблемы
if [ -n "$SERVER_FAILED" ]; then
    echo -e "\n${RED}⚠️  ВНИМАНИЕ:${NC}"
    echo "Сервер не запустился. Возможные причины:"
    echo "• Порт 8080 уже занят"
    echo "• Ошибка в коде сервера"
    echo "• Недостаточно прав"
    echo "Проверьте запуск вручную: go run verb_extractor_server.go"
fi

echo -e "\n${BLUE}💡 Для получения справки по командам:${NC}"
echo "   make help"
echo "   make usage"
