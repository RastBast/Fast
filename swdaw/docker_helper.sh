#!/bin/bash

# 🐳 Автоматический помощник Docker для новичков

set -e

GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m"

echo -e "${BLUE}🐳 DOCKER ПОМОЩНИК ДЛЯ НОВИЧКОВ${NC}"
echo "=================================="

# Проверка Docker
echo -e "\n${YELLOW}🔧 ШАГ 1: ПРОВЕРКА DOCKER${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker не установлен${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose не установлен${NC}"  
    exit 1
fi

echo -e "${GREEN}✅ Docker установлен${NC}"

# Проверка прав
if docker info &> /dev/null; then
    DOCKER_CMD="docker-compose"
    echo -e "${GREEN}✅ Права доступа в порядке${NC}"
else
    DOCKER_CMD="sudo docker-compose"
    echo -e "${YELLOW}⚠️ Будем использовать sudo${NC}"
fi

# Проверка файлов
echo -e "\n${YELLOW}📁 ШАГ 2: ПРОВЕРКА ФАЙЛОВ${NC}"

if [[ -f "proper_verb_server_fixed.go" ]]; then
    echo -e "${GREEN}✅ Go файл найден${NC}"
else
    echo -e "${RED}❌ proper_verb_server_fixed.go не найден${NC}"
    exit 1
fi

# Запуск
echo -e "\n${YELLOW}🚀 ШАГ 3: ЗАПУСК${NC}"
echo "Нажмите Enter для продолжения..."
read

echo -e "${BLUE}Запускаем контейнер...${NC}"
$DOCKER_CMD up --build -d

echo -e "\n${BLUE}Ждем запуска (10 секунд)...${NC}"
sleep 10

# Тест
if curl -f -s http://localhost:8080/api/health > /dev/null; then
    echo -e "${GREEN}🎉 УСПЕХ! Сервер работает!${NC}"
    
    echo -e "\n${BLUE}Тестируем API...${NC}"
    response=$(curl -s -X POST http://localhost:8080/api/extract-verbs \
        -H "Content-Type: application/json" \
        -d '{"text":"Он ударил мяч. Нужно ударить точно. Я ударяю по цели."}')
    
    echo -e "${GREEN}Ответ: $response${NC}"
    
    echo -e "\n${BLUE}💡 Полезные команды:${NC}"
    echo "  Логи:      $DOCKER_CMD logs -f"
    echo "  Остановка: $DOCKER_CMD down" 
    echo "  Статус:    $DOCKER_CMD ps"
    
else
    echo -e "${RED}❌ Сервер не отвечает${NC}"
    $DOCKER_CMD logs
fi
