package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

// HierarchicalNavigation - навигация на основе статичной иерархии меню
// Каждое меню знает своего родителя, как в файловой системе
type HierarchicalNavigation struct {
	hierarchy map[string]string // menu_id -> parent_id
	backBtn   *tele.Btn
}

func NewHierarchicalNavigation() *HierarchicalNavigation {
	selector := &tele.ReplyMarkup{}
	backBtn := selector.Data("⬅️ Назад", "nav_back")

	hn := &HierarchicalNavigation{
		hierarchy: make(map[string]string),
		backBtn:   backBtn,
	}

	// Определяем иерархию меню один раз
	hn.defineMenuHierarchy()

	return hn
}

// defineMenuHierarchy определяет статичную структуру меню
func (hn *HierarchicalNavigation) defineMenuHierarchy() {
	// Просто карта: текущее_меню -> родительское_меню
	hn.hierarchy = map[string]string{
		// Основные разделы
		"channels": "main",
		"stats":    "main",
		"settings": "main",
		"profile":  "main",
		"help":     "main",

		// Подразделы каналов
		"add_channel":    "channels",
		"list_channels":  "channels",
		"remove_channel": "channels",
		"channel_stats":  "channels",

		// Подразделы статистики
		"daily_stats":   "stats",
		"weekly_stats":  "stats",
		"monthly_stats": "stats",
		"export_stats":  "stats",

		// Подразделы настроек
		"language":      "settings",
		"notifications": "settings",
		"theme":         "settings",
		"advanced":      "settings",

		// Подразделы профиля
		"edit_profile":   "profile",
		"view_profile":   "profile",
		"delete_profile": "profile",

		// Глубокая вложенность (пример)
		"lang_russian":   "language",
		"lang_english":   "language",
		"notif_channels": "notifications",
		"notif_stats":    "notifications",
		"theme_dark":     "theme",
		"theme_light":    "theme",

		// Еще глубже
		"notif_channels_new":    "notif_channels",
		"notif_channels_update": "notif_channels",
		"notif_stats_daily":     "notif_stats",
		"notif_stats_weekly":    "notif_stats",
	}
}

// RegisterMenu регистрирует новое меню с родителем
func (hn *HierarchicalNavigation) RegisterMenu(menuID, parentID string) {
	hn.hierarchy[menuID] = parentID
}

// GetParent возвращает родительское меню
func (hn *HierarchicalNavigation) GetParent(menuID string) (string, bool) {
	parent, exists := hn.hierarchy[menuID]
	return parent, exists
}

// GetBackButton возвращает универсальную кнопку "назад"
func (hn *HierarchicalNavigation) GetBackButton() *tele.Btn {
	return hn.backBtn
}

// HasParent проверяет, есть ли у меню родитель
func (hn *HierarchicalNavigation) HasParent(menuID string) bool {
	_, exists := hn.hierarchy[menuID]
	return exists
}

// AddBackButton добавляет кнопку "назад" только если есть родитель
func (hn *HierarchicalNavigation) AddBackButton(keyboard *tele.ReplyMarkup, currentMenu string) {
	if !hn.HasParent(currentMenu) {
		return // Нет родителя - нет кнопки
	}

	if keyboard.InlineKeyboard == nil {
		keyboard.InlineKeyboard = make([][]tele.Btn, 0)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tele.Btn{*hn.backBtn})
}

// GetBreadcrumb возвращает путь до корня (для отладки/показа пути)
func (hn *HierarchicalNavigation) GetBreadcrumb(menuID string) []string {
	var path []string
	current := menuID

	// Идем вверх по иерархии до корня
	for current != "" && current != "main" {
		path = append([]string{current}, path...) // Добавляем в начало
		parent, exists := hn.hierarchy[current]
		if !exists {
			break
		}
		current = parent
	}

	// Добавляем main в начало
	if len(path) > 0 {
		path = append([]string{"main"}, path...)
	}

	return path
}

// CreateMenuButton создает кнопку для перехода в меню
func (hn *HierarchicalNavigation) CreateMenuButton(text, menuID string) *tele.Btn {
	selector := &tele.ReplyMarkup{}
	return selector.Data(text, "menu:"+menuID)
}

// Пример использования
type SimpleBot struct {
	*tele.Bot
	nav *HierarchicalNavigation
}

func NewSimpleBot(token string) (*SimpleBot, error) {
	bot, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	nav := NewHierarchicalNavigation()

	sb := &SimpleBot{
		Bot: bot,
		nav: nav,
	}

	sb.setupHandlers()
	return sb, nil
}

func (sb *SimpleBot) setupHandlers() {
	// Универсальный обработчик кнопки "назад"
	sb.Handle(sb.nav.GetBackButton(), sb.handleBack)

	// Команды
	sb.Handle("/start", sb.handleStart)

	// Обработчик всех меню
	sb.Handle(tele.OnCallback, sb.handleCallback)
}

// handleBack - СУПЕР ПРОСТОЙ обработчик кнопки "назад"
func (sb *SimpleBot) handleBack(c tele.Context) error {
	// Получаем текущее меню из callback query (или из context)
	currentMenu := sb.getCurrentMenu(c)

	// Просто ищем родителя в статичной карте
	parentMenu, hasParent := sb.nav.GetParent(currentMenu)

	if !hasParent {
		return c.Respond(&tele.CallbackResponse{
			Text: "Вы уже в главном меню",
		})
	}

	// Переходим к родительскому меню
	return sb.showMenu(c, parentMenu)
}

// getCurrentMenu определяет текущее меню из контекста
func (sb *SimpleBot) getCurrentMenu(c tele.Context) string {
	// Можно извлечь из callback data предыдущей кнопки
	// Или из message, или хранить в user_data
	// Для простоты будем извлекать из текста сообщения

	text := c.Message().Text
	if text == "" && c.Callback() != nil {
		text = c.Callback().Message.Text
	}

	// Простое определение по тексту (можно улучшить)
	switch {
	case strings.Contains(text, "Каналы"):
		return "channels"
	case strings.Contains(text, "Статистика"):
		return "stats"
	case strings.Contains(text, "Настройки"):
		return "settings"
	case strings.Contains(text, "Профиль"):
		return "profile"
	case strings.Contains(text, "Добавить канал"):
		return "add_channel"
	case strings.Contains(text, "Уведомления"):
		return "notifications"
	case strings.Contains(text, "Язык"):
		return "language"
	case strings.Contains(text, "Тема"):
		return "theme"
	default:
		return "main"
	}
}

func (sb *SimpleBot) handleStart(c tele.Context) error {
	return sb.showMenu(c, "main")
}

func (sb *SimpleBot) handleCallback(c tele.Context) error {
	data := c.Callback().Data

	// Пропускаем кнопку "назад" - она обрабатывается отдельно
	if data == "nav_back" {
		return nil
	}

	// Обрабатываем кнопки меню
	if strings.HasPrefix(data, "menu:") {
		menuID := strings.TrimPrefix(data, "menu:")
		return sb.showMenu(c, menuID)
	}

	return c.Respond()
}

func (sb *SimpleBot) showMenu(c tele.Context, menuID string) error {
	switch menuID {
	case "main":
		return sb.showMainMenu(c)
	case "channels":
		return sb.showChannelsMenu(c)
	case "stats":
		return sb.showStatsMenu(c)
	case "settings":
		return sb.showSettingsMenu(c)
	case "add_channel":
		return sb.showAddChannelMenu(c)
	case "notifications":
		return sb.showNotificationsMenu(c)
	case "language":
		return sb.showLanguageMenu(c)
	default:
		return c.Edit("Неизвестное меню: " + menuID)
	}
}

func (sb *SimpleBot) showMainMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnChannels := sb.nav.CreateMenuButton("📊 Каналы", "channels")
	btnStats := sb.nav.CreateMenuButton("📈 Статистика", "stats")
	btnSettings := sb.nav.CreateMenuButton("⚙️ Настройки", "settings")

	selector.Inline(
		selector.Row(*btnChannels),
		selector.Row(*btnStats, *btnSettings),
	)

	// НЕ добавляем кнопку "назад" в главное меню

	text := "🏠 <b>Главное меню</b>\n\nВыберите раздел:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (sb *SimpleBot) showChannelsMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnAdd := sb.nav.CreateMenuButton("➕ Добавить канал", "add_channel")
	btnList := sb.nav.CreateMenuButton("📋 Список каналов", "list_channels")
	btnStats := sb.nav.CreateMenuButton("📊 Статистика каналов", "channel_stats")

	selector.Inline(
		selector.Row(*btnAdd),
		selector.Row(*btnList, *btnStats),
	)

	// Автоматически добавляем кнопку "назад"
	sb.nav.AddBackButton(selector, "channels")

	breadcrumb := sb.nav.GetBreadcrumb("channels")
	text := fmt.Sprintf("📊 <b>Управление каналами</b>\n\n📍 Путь: %v\n\nВыберите действие:", breadcrumb)

	return c.Edit(text, selector, tele.ModeHTML)
}

func (sb *SimpleBot) showSettingsMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnLang := sb.nav.CreateMenuButton("🌐 Язык", "language")
	btnNotif := sb.nav.CreateMenuButton("🔔 Уведомления", "notifications")
	btnTheme := sb.nav.CreateMenuButton("🎨 Тема", "theme")

	selector.Inline(
		selector.Row(*btnLang, *btnNotif),
		selector.Row(*btnTheme),
	)

	sb.nav.AddBackButton(selector, "settings")

	text := "⚙️ <b>Настройки</b>\n\nВыберите параметр:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (sb *SimpleBot) showAddChannelMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnByLink := selector.Data("🔗 По ссылке", "add_by_link")
	btnByUsername := selector.Data("👤 По username", "add_by_username")

	selector.Inline(
		selector.Row(*btnByLink),
		selector.Row(*btnByUsername),
	)

	sb.nav.AddBackButton(selector, "add_channel")

	breadcrumb := sb.nav.GetBreadcrumb("add_channel")
	text := fmt.Sprintf("➕ <b>Добавление канала</b>\n\n📍 Путь: %v\n\nВыберите способ:", breadcrumb)

	return c.Edit(text, selector, tele.ModeHTML)
}

func (sb *SimpleBot) showNotificationsMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnChannels := sb.nav.CreateMenuButton("📊 Уведомления о каналах", "notif_channels")
	btnStats := sb.nav.CreateMenuButton("📈 Уведомления о статистике", "notif_stats")

	selector.Inline(
		selector.Row(*btnChannels),
		selector.Row(*btnStats),
	)

	sb.nav.AddBackButton(selector, "notifications")

	text := "🔔 <b>Настройки уведомлений</b>\n\nВыберите тип:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (sb *SimpleBot) showLanguageMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnRu := sb.nav.CreateMenuButton("🇷🇺 Русский", "lang_russian")
	btnEn := sb.nav.CreateMenuButton("🇺🇸 English", "lang_english")
	btnDe := sb.nav.CreateMenuButton("🇩🇪 Deutsch", "lang_german")

	selector.Inline(
		selector.Row(*btnRu, *btnEn),
		selector.Row(*btnDe),
	)

	sb.nav.AddBackButton(selector, "language")

	breadcrumb := sb.nav.GetBreadcrumb("language")
	text := fmt.Sprintf("🌐 <b>Выбор языка</b>\n\n📍 Путь: %v\n\nВыберите язык:", breadcrumb)

	return c.Edit(text, selector, tele.ModeHTML)
}
