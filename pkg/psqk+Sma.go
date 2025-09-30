package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
	tele "gopkg.in/telebot.v3"
)

// PersistentNavigationManager - менеджер с сохранением в PostgreSQL
type PersistentNavigationManager struct {
	db       *sql.DB
	cache    map[int64][]string  // Кэш для быстрого доступа
	cacheTTL map[int64]time.Time // TTL для элементов кэша
	mutex    sync.RWMutex
	backBtn  *tele.Btn

	// Настройки оптимизации
	maxCacheSize    int
	cacheTimeout    time.Duration
	maxStackDepth   int
	cleanupInterval time.Duration
}

// NavigationState для сериализации в JSON
type NavigationState struct {
	UserID    int64     `json:"user_id"`
	MenuStack []string  `json:"menu_stack"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewPersistentNavigationManager(db *sql.DB) *PersistentNavigationManager {
	selector := &tele.ReplyMarkup{}
	backBtn := selector.Data("⬅️ Назад", "persistent_back")

	pnm := &PersistentNavigationManager{
		db:              db,
		cache:           make(map[int64][]string),
		cacheTTL:        make(map[int64]time.Time),
		backBtn:         backBtn,
		maxCacheSize:    1000, // Кэшируем только 1000 активных пользователей
		cacheTimeout:    10 * time.Minute,
		maxStackDepth:   20,
		cleanupInterval: 30 * time.Minute,
	}

	// Создаем таблицу
	pnm.createTable()

	// Запускаем фоновые процессы
	go pnm.cacheCleanupRoutine()
	go pnm.dbCleanupRoutine()

	return pnm
}

// createTable создает таблицу для навигации
func (pnm *PersistentNavigationManager) createTable() {
	query := `
    CREATE TABLE IF NOT EXISTS user_navigation (
        user_id BIGINT PRIMARY KEY,
        menu_stack JSONB NOT NULL,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Индекс для быстрой очистки старых записей
    CREATE INDEX IF NOT EXISTS idx_user_navigation_updated_at 
    ON user_navigation(updated_at);
    `

	_, err := pnm.db.Exec(query)
	if err != nil {
		log.Printf("❌ Ошибка создания таблицы навигации: %v", err)
	} else {
		log.Println("✅ Таблица навигации готова")
	}
}

// PushMenu добавляет меню с гибридным подходом (кэш + БД)
func (pnm *PersistentNavigationManager) PushMenu(userID int64, menuID string) error {
	pnm.mutex.Lock()
	defer pnm.mutex.Unlock()

	// Получаем текущий стек (из кэша или БД)
	stack, err := pnm.getStackFromCacheOrDB(userID)
	if err != nil {
		return err
	}

	// Проверяем дубликаты
	if len(stack) > 0 && stack[len(stack)-1] == menuID {
		return nil
	}

	// Ограничиваем глубину стека
	if len(stack) >= pnm.maxStackDepth {
		// Удаляем старые элементы
		keepSize := pnm.maxStackDepth - 5
		stack = stack[len(stack)-keepSize:]
		log.Printf("⚠️  User %d: Stack trimmed to %d elements", userID, keepSize)
	}

	// Добавляем новое меню
	stack = append(stack, menuID)

	// Обновляем кэш
	pnm.cache[userID] = stack
	pnm.cacheTTL[userID] = time.Now().Add(pnm.cacheTimeout)

	// Асинхронно сохраняем в БД (не блокируем пользователя)
	go pnm.saveToDBAsync(userID, stack)

	return nil
}

// PopMenu убирает последнее меню
func (pnm *PersistentNavigationManager) PopMenu(userID int64) (string, bool, error) {
	pnm.mutex.Lock()
	defer pnm.mutex.Unlock()

	stack, err := pnm.getStackFromCacheOrDB(userID)
	if err != nil {
		return "", false, err
	}

	if len(stack) <= 1 {
		return "", false, nil
	}

	// Убираем последний элемент
	stack = stack[:len(stack)-1]
	prevMenu := stack[len(stack)-1]

	// Обновляем кэш
	pnm.cache[userID] = stack
	pnm.cacheTTL[userID] = time.Now().Add(pnm.cacheTimeout)

	// Асинхронно сохраняем в БД
	go pnm.saveToDBAsync(userID, stack)

	return prevMenu, true, nil
}

// getStackFromCacheOrDB получает стек из кэша или БД
func (pnm *PersistentNavigationManager) getStackFromCacheOrDB(userID int64) ([]string, error) {
	// Проверяем кэш
	if stack, exists := pnm.cache[userID]; exists {
		if ttl, hasTTL := pnm.cacheTTL[userID]; hasTTL && time.Now().Before(ttl) {
			return stack, nil
		}
		// TTL истек, удаляем из кэша
		delete(pnm.cache, userID)
		delete(pnm.cacheTTL, userID)
	}

	// Загружаем из БД
	return pnm.loadFromDB(userID)
}

// loadFromDB загружает навигацию из БД
func (pnm *PersistentNavigationManager) loadFromDB(userID int64) ([]string, error) {
	var stackJSON []byte
	query := "SELECT menu_stack FROM user_navigation WHERE user_id = $1"

	err := pnm.db.QueryRow(query, userID).Scan(&stackJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return []string{}, nil // Новый пользователь
		}
		return nil, err
	}

	var stack []string
	err = json.Unmarshal(stackJSON, &stack)
	if err != nil {
		return nil, err
	}

	// Добавляем в кэш после загрузки
	pnm.cache[userID] = stack
	pnm.cacheTTL[userID] = time.Now().Add(pnm.cacheTimeout)

	return stack, nil
}

// saveToDBAsync асинхронно сохраняет в БД
func (pnm *PersistentNavigationManager) saveToDBAsync(userID int64, stack []string) {
	stackJSON, err := json.Marshal(stack)
	if err != nil {
		log.Printf("❌ Ошибка сериализации стека для user %d: %v", userID, err)
		return
	}

	query := `
    INSERT INTO user_navigation (user_id, menu_stack, updated_at) 
    VALUES ($1, $2, CURRENT_TIMESTAMP) 
    ON CONFLICT (user_id) 
    DO UPDATE SET 
        menu_stack = EXCLUDED.menu_stack,
        updated_at = CURRENT_TIMESTAMP
    `

	_, err = pnm.db.Exec(query, userID, stackJSON)
	if err != nil {
		log.Printf("❌ Ошибка сохранения навигации для user %d: %v", userID, err)
	}
}

// cacheCleanupRoutine очищает кэш от устаревших данных
func (pnm *PersistentNavigationManager) cacheCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pnm.cleanupCache()
		}
	}
}

// cleanupCache удаляет устаревшие элементы из кэша
func (pnm *PersistentNavigationManager) cleanupCache() {
	pnm.mutex.Lock()
	defer pnm.mutex.Unlock()

	now := time.Now()
	var cleaned int

	for userID, ttl := range pnm.cacheTTL {
		if now.After(ttl) {
			delete(pnm.cache, userID)
			delete(pnm.cacheTTL, userID)
			cleaned++
		}
	}

	// Если кэш слишком большой, удаляем самые старые элементы
	if len(pnm.cache) > pnm.maxCacheSize {
		// Простая стратегия: удаляем случайные элементы
		excess := len(pnm.cache) - pnm.maxCacheSize + 100 // +100 для буфера
		for userID := range pnm.cache {
			if excess <= 0 {
				break
			}
			delete(pnm.cache, userID)
			delete(pnm.cacheTTL, userID)
			excess--
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Printf("🧹 Очищено %d элементов из кэша навигации", cleaned)
	}
}

// dbCleanupRoutine удаляет старые записи из БД
func (pnm *PersistentNavigationManager) dbCleanupRoutine() {
	ticker := time.NewTicker(pnm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pnm.cleanupOldNavigationData(24 * time.Hour) // Удаляем данные старше суток
		}
	}
}

// cleanupOldNavigationData удаляет старые данные навигации
func (pnm *PersistentNavigationManager) cleanupOldNavigationData(maxAge time.Duration) {
	query := "DELETE FROM user_navigation WHERE updated_at < $1"
	cutoff := time.Now().Add(-maxAge)

	result, err := pnm.db.Exec(query, cutoff)
	if err != nil {
		log.Printf("❌ Ошибка очистки старых данных навигации: %v", err)
		return
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("🧹 Удалено %d старых записей навигации", affected)
	}
}

// GetNavigationStats возвращает статистику
func (pnm *PersistentNavigationManager) GetNavigationStats() (map[string]interface{}, error) {
	pnm.mutex.RLock()
	cacheSize := len(pnm.cache)
	pnm.mutex.RUnlock()

	// Статистика из БД
	var dbRecords int
	var avgDepth float64
	var maxDepth int

	query := `
    SELECT 
        COUNT(*) as total_records,
        AVG(jsonb_array_length(menu_stack)) as avg_depth,
        MAX(jsonb_array_length(menu_stack)) as max_depth
    FROM user_navigation
    `

	err := pnm.db.QueryRow(query).Scan(&dbRecords, &avgDepth, &maxDepth)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"cache_size":        cacheSize,
		"max_cache_size":    pnm.maxCacheSize,
		"db_records":        dbRecords,
		"average_depth":     avgDepth,
		"max_depth":         maxDepth,
		"max_allowed_depth": pnm.maxStackDepth,
		"cache_timeout_min": pnm.cacheTimeout.Minutes(),
	}, nil
}

// GetBackButton возвращает кнопку назад
func (pnm *PersistentNavigationManager) GetBackButton() *tele.Btn {
	return pnm.backBtn
}

// AddBackButton добавляет кнопку к клавиатуре
func (pnm *PersistentNavigationManager) AddBackButton(keyboard *tele.ReplyMarkup) {
	if keyboard.InlineKeyboard == nil {
		keyboard.InlineKeyboard = make([][]tele.Btn, 0)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tele.Btn{*pnm.backBtn})
}
