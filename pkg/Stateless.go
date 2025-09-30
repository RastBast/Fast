package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	tele "gopkg.in/telebot.v3"
)

// StatelessNavigationManager - навигация без сохранения состояния
// Путь кодируется прямо в callback_data кнопки "назад"
type StatelessNavigationManager struct {
	backBtnPrefix string
	maxPathLength int // Ограничение длины пути в символах
}

// NavigationPath представляет путь навигации
type NavigationPath struct {
	Path []string `json:"p"`
}

func NewStatelessNavigationManager() *StatelessNavigationManager {
	return &StatelessNavigationManager{
		backBtnPrefix: "back:",
		maxPathLength: 200, // Максимум 200 символов для Telegram callback_data
	}
}

// CreateBackButton создает кнопку "назад" с закодированным путем
func (snm *StatelessNavigationManager) CreateBackButton(currentPath []string) *tele.Btn {
	if len(currentPath) <= 1 {
		return nil // Нет предыдущего уровня
	}

	// Убираем текущий уровень, оставляем предыдущий путь
	prevPath := currentPath[:len(currentPath)-1]

	// Кодируем путь в callback_data
	encodedPath := snm.encodePath(prevPath)
	callbackData := snm.backBtnPrefix + encodedPath

	// Проверяем ограничение Telegram (64 символа для callback_data)
	if len(callbackData) > 64 {
		// Используем хэш для длинных путей
		hash := snm.hashPath(prevPath)
		callbackData = snm.backBtnPrefix + "h:" + hash
	}

	selector := &tele.ReplyMarkup{}
	return selector.Data("⬅️ Назад", callbackData)
}

// AddBackButton добавляет кнопку "назад" к клавиатуре
func (snm *StatelessNavigationManager) AddBackButton(keyboard *tele.ReplyMarkup, currentPath []string) {
	backBtn := snm.CreateBackButton(currentPath)
	if backBtn == nil {
		return // Нет кнопки назад для корневого уровня
	}

	if keyboard.InlineKeyboard == nil {
		keyboard.InlineKeyboard = make([][]tele.Btn, 0)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tele.Btn{*backBtn})
}

// IsBackButton проверяет, является ли callback кнопкой "назад"
func (snm *StatelessNavigationManager) IsBackButton(callbackData string) bool {
	return strings.HasPrefix(callbackData, snm.backBtnPrefix)
}

// DecodeBackButton декодирует путь из callback кнопки "назад"
func (snm *StatelessNavigationManager) DecodeBackButton(callbackData string) ([]string, error) {
	if !snm.IsBackButton(callbackData) {
		return nil, fmt.Errorf("not a back button")
	}

	encodedPath := strings.TrimPrefix(callbackData, snm.backBtnPrefix)

	// Проверяем, это хэш или обычный путь
	if strings.HasPrefix(encodedPath, "h:") {
		// Это хэшированный путь - возвращаем к главному меню
		return []string{"main"}, nil
	}

	return snm.decodePath(encodedPath)
}

// encodePath кодирует путь в строку
func (snm *StatelessNavigationManager) encodePath(path []string) string {
	if len(path) == 0 {
		return ""
	}

	pathData := NavigationPath{Path: path}
	jsonData, err := json.Marshal(pathData)
	if err != nil {
		log.Printf("❌ Ошибка кодирования пути: %v", err)
		return ""
	}

	// Кодируем в base64 для безопасной передачи
	encoded := base64.URLEncoding.EncodeToString(jsonData)

	// Обрезаем если слишком длинно
	if len(encoded) > snm.maxPathLength {
		encoded = encoded[:snm.maxPathLength]
	}

	return encoded
}

// decodePath декодирует путь из строк��
func (snm *StatelessNavigationManager) decodePath(encoded string) ([]string, error) {
	if encoded == "" {
		return []string{}, nil
	}

	jsonData, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %v", err)
	}

	var pathData NavigationPath
	err = json.Unmarshal(jsonData, &pathData)
	if err != nil {
		return nil, fmt.Errorf("invalid json: %v", err)
	}

	return pathData.Path, nil
}

// hashPath создает хэш длинного пути
func (snm *StatelessNavigationManager) hashPath(path []string) string {
	pathStr := strings.Join(path, "|")
	hash := md5.Sum([]byte(pathStr))
	return base64.URLEncoding.EncodeToString(hash[:])[:8] // Берем первые 8 символов
}

// CreateMenuButton создает кнопку меню с кодированием следующего пути
func (snm *StatelessNavigationManager) CreateMenuButton(text, menuID string, currentPath []string) *tele.Btn {
	// Создаем новый путь с добавлением текущего меню
	newPath := append(currentPath, menuID)

	selector := &tele.ReplyMarkup{}

	// В callback_data кодируем информацию о переходе
	// Формат: "menu:menuID:encodedCurrentPath"
	encodedCurrentPath := snm.encodePath(currentPath)
	callbackData := fmt.Sprintf("menu:%s:%s", menuID, encodedCurrentPath)

	// Проверяем ограничение длины
	if len(callbackData) > 64 {
		// Используем сокращенный формат
		callbackData = fmt.Sprintf("menu:%s", menuID)
	}

	return selector.Data(text, callbackData)
}

// DecodeMenuButton декодирует информацию из кнопки меню
func (snm *StatelessNavigationManager) DecodeMenuButton(callbackData string) (menuID string, currentPath []string, err error) {
	if !strings.HasPrefix(callbackData, "menu:") {
		return "", nil, fmt.Errorf("not a menu button")
	}

	parts := strings.SplitN(callbackData, ":", 3)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("invalid menu button format")
	}

	menuID = parts[1]

	if len(parts) == 3 && parts[2] != "" {
		currentPath, err = snm.decodePath(parts[2])
		if err != nil {
			// Если не удалось декодировать, используем пустой путь
			currentPath = []string{}
			err = nil
		}
	} else {
		currentPath = []string{}
	}

	return menuID, currentPath, nil
}
