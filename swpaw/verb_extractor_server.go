package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// VerbResponse представляет ответ API с глаголами в начальной форме
type VerbResponse struct {
	Verbs []string `json:"verbs"`
	Count int      `json:"count"`
	Time  string   `json:"processing_time"`
}

// ErrorResponse представляет структуру ошибки API
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TextRequest представляет входящий запрос с текстом
type TextRequest struct {
	Text string `json:"text"`
}

// VerbExtractor - основная структура для извлечения глаголов
type VerbExtractor struct {
	verbRules map[string]string
	endings   []string
	prefixes  []string
	stopWords map[string]bool
	verbForms map[string]string
	logger    *log.Logger
}

// NewVerbExtractor создает новый экземпляр VerbExtractor
func NewVerbExtractor() *VerbExtractor {
	ve := &VerbExtractor{
		verbRules: make(map[string]string),
		stopWords: make(map[string]bool),
		verbForms: make(map[string]string),
		logger:    log.New(os.Stdout, "[VERB_EXTRACTOR] ", log.LstdFlags),
	}

	ve.initializeVerbDatabase()
	ve.initializeStopWords()
	return ve
}

// initializeVerbDatabase инициализирует базу данных глаголов и правил
func (ve *VerbExtractor) initializeVerbDatabase() {
	// Словарь базовых глаголов и их форм (упрощенная версия)
	basicVerbs := map[string]string{
		"бежать": "бежать", "бежит": "бежать", "бежал": "бежать", "беги": "бежать",
		"видеть": "видеть", "видит": "видеть", "видел": "видеть", "смотри": "смотреть",
		"говорить": "говорить", "говорит": "говорить", "сказать": "сказать", "скажи": "сказать",
		"делать": "делать", "делает": "делать", "делал": "делать", "сделать": "сделать",
		"думать": "думать", "думает": "думать", "думал": "думать", "подумать": "подумать",
		"жить": "жить", "живет": "жить", "жил": "жить", "прожить": "прожить",
		"знать": "знать", "знает": "знать", "знал": "знать", "узнать": "узнать",
		"идти": "идти", "идет": "идти", "шел": "идти", "пойти": "пойти",
		"играть": "играть", "играет": "играть", "играл": "играть", "сыграть": "сыграть",
		"любить": "любить", "любит": "любить", "любил": "любить", "полюбить": "полюбить",
		"мочь": "мочь", "может": "мочь", "мог": "мочь", "смочь": "смочь",
		"писать": "писать", "пишет": "писать", "писал": "писать", "написать": "написать",
		"работать": "работать", "работает": "работать", "работал": "работать",
		"читать": "читать", "читает": "читать", "читал": "читать", "прочитать": "прочитать",
		"ударить": "ударить", "ударял": "ударить", "ударяю": "ударить", "ударил": "ударить",
		"учиться": "учиться", "учится": "учиться", "учился": "учиться", "выучиться": "выучиться",
		"хотеть": "хотеть", "хочет": "хотеть", "хотел": "хотеть", "захотеть": "захотеть",
	}

	for form, lemma := range basicVerbs {
		ve.verbForms[strings.ToLower(form)] = lemma
	}

	// Окончания глаголов для определения
	ve.endings = []string{
		"ить", "ать", "ять", "еть", "оть", "уть", "ыть",
		"ет", "ит", "ют", "ят", "ишь", "ешь", "им", "ем",
		"ил", "ла", "ло", "ли", "ал", "ел", "ол", "ул",
		"ся", "сь", "тся", "ться",
	}

	// Префиксы глаголов
	ve.prefixes = []string{
		"за", "по", "до", "от", "под", "пере", "при", "про", "раз", "с", "у", "в", "вы", "на", "об",
	}
}

// initializeStopWords инициализирует стоп-слова
func (ve *VerbExtractor) initializeStopWords() {
	stopWordsList := []string{
		"и", "в", "на", "с", "по", "для", "от", "к", "у", "о", "об", "за", "под", "над", "при",
		"а", "но", "или", "да", "нет", "не", "ни", "что", "как", "где", "когда", "почему",
		"он", "она", "оно", "они", "я", "ты", "мы", "вы", "его", "ее", "их", "мой", "твой",
		"это", "то", "все", "всё", "ещё", "уже", "только", "очень", "так", "там", "здесь",
	}

	for _, word := range stopWordsList {
		ve.stopWords[word] = true
	}
}

// isVerb проверяет, является ли слово глаголом
func (ve *VerbExtractor) isVerb(word string) bool {
	word = strings.ToLower(word)

	// Проверка в словаре известных глаголов
	if _, exists := ve.verbForms[word]; exists {
		return true
	}

	// Эвристическая проверка по окончаниям
	for _, ending := range ve.endings {
		if strings.HasSuffix(word, ending) && len(word) > len(ending)+1 {
			return true
		}
	}

	return false
}

// lemmatizeVerb приводит глагол к начальной форме
func (ve *VerbExtractor) lemmatizeVerb(word string) string {
	word = strings.ToLower(word)

	// Проверка в словаре
	if lemma, exists := ve.verbForms[word]; exists {
		return lemma
	}

	// Упрощенная лемматизация для неизвестных глаголов
	// Удаляем возвратные окончания
	word = strings.TrimSuffix(word, "ся")
	word = strings.TrimSuffix(word, "сь")

	// Простейшая нормализация окончаний
	if strings.HasSuffix(word, "ет") || strings.HasSuffix(word, "ит") {
		word = word[:len(word)-2] + "ить"
	} else if strings.HasSuffix(word, "ал") || strings.HasSuffix(word, "ел") {
		word = word[:len(word)-2] + "ать"
	}

	return word
}

// cleanText очищает и нормализует текст
func (ve *VerbExtractor) cleanText(text string) string {
	// Удаляем лишние пробелы и приводим к нижнему регистру
	text = strings.ToLower(text)

	// Заменяем знаки пунктуации на пробелы
	reg := regexp.MustCompile(`[^\p{L}\s]+`)
	text = reg.ReplaceAllString(text, " ")

	// Удаляем множественные пробелы
	reg = regexp.MustCompile(`\s+`)
	text = reg.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// ExtractVerbs извлекает все глаголы из текста и приводит их к начальной форме
func (ve *VerbExtractor) ExtractVerbs(text string) []string {
	start := time.Now()

	// Очищаем текст
	cleanedText := ve.cleanText(text)
	words := strings.Fields(cleanedText)

	verbMap := make(map[string]bool)
	var verbs []string

	for _, word := range words {
		// Пропускаем стоп-слова и слишком короткие слова
		if ve.stopWords[word] || len(word) < 3 {
			continue
		}

		// Проверяем, является ли слово глаголом
		if ve.isVerb(word) {
			lemma := ve.lemmatizeVerb(word)
			if !verbMap[lemma] {
				verbMap[lemma] = true
				verbs = append(verbs, lemma)
			}
		}
	}

	ve.logger.Printf("Extracted %d unique verbs in %v", len(verbs), time.Since(start))
	return verbs
}

// VerbServer представляет HTTP сервер
type VerbServer struct {
	extractor *VerbExtractor
	logger    *log.Logger
}

// NewVerbServer создает новый экземпляр сервера
func NewVerbServer() *VerbServer {
	return &VerbServer{
		extractor: NewVerbExtractor(),
		logger:    log.New(os.Stdout, "[SERVER] ", log.LstdFlags),
	}
}

// validateRequest валидирует входящий запрос
func (vs *VerbServer) validateRequest(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method not allowed: %s", r.Method)
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	return nil
}

// parseRequest парсит JSON запрос
func (vs *VerbServer) parseRequest(r *http.Request) (*TextRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	var req TextRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if len(strings.TrimSpace(req.Text)) == 0 {
		return nil, fmt.Errorf("text field is empty")
	}

	return &req, nil
}

// sendErrorResponse отправляет ошибку в формате JSON
func (vs *VerbServer) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Code:    statusCode,
		Message: message,
	}

	json.NewEncoder(w).Encode(errorResp)
}

// sendSuccessResponse отправляет успешный ответ
func (vs *VerbServer) sendSuccessResponse(w http.ResponseWriter, verbs []string, processingTime time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := VerbResponse{
		Verbs: verbs,
		Count: len(verbs),
		Time:  processingTime.String(),
	}

	json.NewEncoder(w).Encode(response)
}

// extractVerbsHandler обрабатывает запросы на извлечение глаголов
func (vs *VerbServer) extractVerbsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Добавляем CORS заголовки
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Обрабатываем preflight запрос
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Валидация запроса
	if err := vs.validateRequest(r); err != nil {
		vs.logger.Printf("Validation error: %v", err)
		vs.sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Парсинг запроса
	req, err := vs.parseRequest(r)
	if err != nil {
		vs.logger.Printf("Parse error: %v", err)
		vs.sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	vs.logger.Printf("Processing text with %d characters", len(req.Text))

	// Извлечение глаголов
	verbs := vs.extractor.ExtractVerbs(req.Text)

	// Отправка ответа
	vs.sendSuccessResponse(w, verbs, time.Since(start))
}

// healthHandler обрабатывает запросы проверки здоровья сервиса
func (vs *VerbServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	json.NewEncoder(w).Encode(health)
}

// metricsHandler предоставляет базовые метрики
func (vs *VerbServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	metrics := map[string]interface{}{
		"verb_database_size": len(vs.extractor.verbForms),
		"stopwords_count":    len(vs.extractor.stopWords),
		"uptime":             time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(metrics)
}

// setupRoutes настраивает маршруты HTTP сервера
func (vs *VerbServer) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/extract-verbs", vs.extractVerbsHandler)
	mux.HandleFunc("/api/health", vs.healthHandler)
	mux.HandleFunc("/api/metrics", vs.metricsHandler)

	// Корневой маршрут с документацией
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, getDocumentationHTML())
	})

	return mux
}

// getDocumentationHTML возвращает HTML документацию API
func getDocumentationHTML() string {
	return `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verb Extractor API</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .endpoint { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .method { color: #2196F3; font-weight: bold; }
        pre { background: #282c34; color: #abb2bf; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .example { margin-top: 10px; }
    </style>
</head>
<body>
    <h1>Verb Extractor API</h1>
    <p>API для извлечения глаголов в начальной форме из русского текста</p>

    <div class="endpoint">
        <h3><span class="method">POST</span> /api/extract-verbs</h3>
        <p>Извлекает все глаголы из текста и приводит их к начальной форме</p>
        <h4>Пример запроса:</h4>
        <pre>{
  "text": "Гарри читал книгу и думал о магии. Он хотел выучить заклинания."
}</pre>
        <h4>Пример ответа:</h4>
        <pre>{
  "verbs": ["читать", "думать", "хотеть", "выучить"],
  "count": 4,
  "processing_time": "1.2345ms"
}</pre>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> /api/health</h3>
        <p>Проверка состояния сервиса</p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> /api/metrics</h3>
        <p>Базовые метрики системы</p>
    </div>

    <h2>Тестирование с curl:</h2>
    <pre>curl -X POST http://localhost:8080/api/extract-verbs      -H "Content-Type: application/json"      -d '{"text":"Я иду в магазин и покупаю хлеб"}'</pre>
</body>
</html>`
}

// gracefulShutdown обрабатывает graceful shutdown сервера
func (vs *VerbServer) gracefulShutdown(server *http.Server) {
	// В реальном приложении здесь был бы обработчик сигналов
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	vs.logger.Println("Server is ready to handle requests at :8080")
}

// Start запускает HTTP сервер
func (vs *VerbServer) Start(addr string) error {
	mux := vs.setupRoutes()

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	vs.gracefulShutdown(server)

	return server.ListenAndServe()
}

// loadTestData загружает тестовые данные из файла
func loadTestData(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString(" ")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return content.String(), nil
}

// createTestData создает тестовый файл с фрагментом из ГПМРМ
func createTestData() error {
	testText := `
	Петуния вышла замуж не за Дурсля, а за университетского профессора, и Гарри попал в гораздо более благоприятную среду. 
	У него были частные учителя, дискуссии с отцом, а главное — книги, сотни и тысячи научных и фантастических книг. 
	В 11 лет Гарри знаком с квантовой механикой, когнитивной психологией, теорией вероятностей и другими вещами. 
	Но Гарри не просто вундеркинд, у него есть загадочная Тёмная сторона, которая явно накладывает свой отпечаток на его мышление.
	Гарри изучал науку и читал книги. Он думал о мире и хотел понять его устройство. 
	Профессор учил его логике и критическому мышлению. Мальчик анализировал данные и проводил эксперименты.
	`

	return os.WriteFile("test_data.txt", []byte(testText), 0o644)
}

func main() {
	// Создаем тестовые данные
	if err := createTestData(); err != nil {
		log.Printf("Warning: Could not create test data: %v", err)
	}

	// Создаем и запускаем сервер
	server := NewVerbServer()

	log.Println("Starting Verb Extractor Server...")
	log.Println("Visit http://localhost:8080 for API documentation")

	if err := server.Start(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
