package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

// UltraSimpleNavigation - максимально простая навигация
// Принцип: каждая кнопка знает куда она ведет назад
type UltraSimpleNavigation struct {
	// Больше ничего не нужно!
}

func NewUltraSimpleNavigation() *UltraSimpleNavigation {
	return &UltraSimpleNavigation{}
}

// CreateBackButton создает кнопку "назад" с указанием куда вернуться
func (usn *UltraSimpleNavigation) CreateBackButton(returnTo string) *tele.Btn {
	selector := &tele.ReplyMarkup{}
	return selector.Data("⬅️ Назад", "back_to:"+returnTo)
}

// CreateMenuButton создает кнопку перехода в меню
func (usn *UltraSimpleNavigation) CreateMenuButton(text, menuID string) *tele.Btn {
	selector := &tele.ReplyMarkup{}
	return selector.Data(text, "goto:"+menuID)
}

// AddBackButton добавляет кнопку "назад" с указанием куда возвращаться
func (usn *UltraSimpleNavigation) AddBackButton(keyboard *tele.ReplyMarkup, returnTo string) {
	if returnTo == "" || returnTo == "main" {
		return // Не добавляем кнопку в главное меню
	}

	backBtn := usn.CreateBackButton(returnTo)
	if keyboard.InlineKeyboard == nil {
		keyboard.InlineKeyboard = make([][]tele.Btn, 0)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tele.Btn{*backBtn})
}

// IsBackButton проверяет, является ли это кнопкой "назад"
func (usn *UltraSimpleNavigation) IsBackButton(callbackData string) (bool, string) {
	if strings.HasPrefix(callbackData, "back_to:") {
		returnTo := strings.TrimPrefix(callbackData, "back_to:")
		return true, returnTo
	}
	return false, ""
}

// IsMenuButton проверяет, является ли это кнопкой меню
func (usn *UltraSimpleNavigation) IsMenuButton(callbackData string) (bool, string) {
	if strings.HasPrefix(callbackData, "goto:") {
		menuID := strings.TrimPrefix(callbackData, "goto:")
		return true, menuID
	}
	return false, ""
}

// Пример использования - СУПЕР ПРОСТОЙ БОТ
type UltraBot struct {
	*tele.Bot
	nav *UltraSimpleNavigation
}

func NewUltraBot(token string) (*UltraBot, error) {
	bot, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	ub := &UltraBot{
		Bot: bot,
		nav: NewUltraSimpleNavigation(),
	}

	ub.setupHandlers()
	return ub, nil
}

func (ub *UltraBot) setupHandlers() {
	ub.Handle("/start", ub.handleStart)
	ub.Handle(tele.OnCallback, ub.handleCallback)
}

func (ub *UltraBot) handleStart(c tele.Context) error {
	return ub.showMainMenu(c)
}

func (ub *UltraBot) handleCallback(c tele.Context) error {
	data := c.Callback().Data

	// Проверяем кнопку "назад"
	if isBack, returnTo := ub.nav.IsBackButton(data); isBack {
		return ub.showMenu(c, returnTo)
	}

	// Проверяем кнопку меню
	if isMenu, menuID := ub.nav.IsMenuButton(data); isMenu {
		return ub.showMenu(c, menuID)
	}

	return c.Respond()
}

func (ub *UltraBot) showMenu(c tele.Context, menuID string) error {
	switch menuID {
	case "main":
		return ub.showMainMenu(c)
	case "channels":
		return ub.showChannelsMenu(c)
	case "settings":
		return ub.showSettingsMenu(c)
	case "add_channel":
		return ub.showAddChannelMenu(c)
	case "language":
		return ub.showLanguageMenu(c)
	default:
		return c.Edit("Неизвестное меню")
	}
}

func (ub *UltraBot) showMainMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnChannels := ub.nav.CreateMenuButton("📊 Каналы", "channels")
	btnSettings := ub.nav.CreateMenuButton("⚙️ Настройки", "settings")

	selector.Inline(
		selector.Row(*btnChannels),
		selector.Row(*btnSettings),
	)

	text := "🏠 <b>Главное меню</b>\n\nВыберите раздел:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (ub *UltraBot) showChannelsMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnAdd := ub.nav.CreateMenuButton("➕ Добавить", "add_channel")
	btnList := ub.nav.CreateMenuButton("📋 Список", "list_channels")

	selector.Inline(
		selector.Row(*btnAdd, *btnList),
	)

	// Говорим кнопке "назад" куда возвращаться
	ub.nav.AddBackButton(selector, "main")

	text := "📊 <b>Управление каналами</b>\n\nВыберите действие:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (ub *UltraBot) showSettingsMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btnLang := ub.nav.CreateMenuButton("🌐 Язык", "language")
	btnNotif := ub.nav.CreateMenuButton("🔔 Уведомления", "notifications")

	selector.Inline(
		selector.Row(*btnLang, *btnNotif),
	)

	ub.nav.AddBackButton(selector, "main")

	text := "⚙️ <b>Настройки</b>\n\nВыберите параметр:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (ub *UltraBot) showAddChannelMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btn1 := selector.Data("🔗 По ссылке", "add_by_link")
	btn2 := selector.Data("👤 По username", "add_by_username")

	selector.Inline(
		selector.Row(*btn1),
		selector.Row(*btn2),
	)

	// Возвращаемся в меню каналов
	ub.nav.AddBackButton(selector, "channels")

	text := "➕ <b>Добавление канала</b>\n\nВыберите способ:"
	return c.Edit(text, selector, tele.ModeHTML)
}

func (ub *UltraBot) showLanguageMenu(c tele.Context) error {
	selector := &tele.ReplyMarkup{}

	btn1 := selector.Data("🇷🇺 Русский", "set_lang_ru")
	btn2 := selector.Data("🇺🇸 English", "set_lang_en")

	selector.Inline(
		selector.Row(*btn1, *btn2),
	)

	// Возвращаемся в настройки
	ub.nav.AddBackButton(selector, "settings")

	text := "🌐 <b>Выбор языка</b>\n\nВыберите язык:"
	return c.Edit(text, selector, tele.ModeHTML)
}
