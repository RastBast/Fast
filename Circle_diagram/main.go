package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Circle struct {
	X, Y    int
	Radius  int
	IsSpawn bool // true для spawn кругов, false для bedroom кругов
}

type MapGenerator struct {
	Width, Height int
	SpawnCount    int
	BedroomCount  int
	SpawnRadius   int
	BedroomRadius int
	MaxGap        int
	CellSize      int
}

// Проверка пересечения кругов
func (c1 Circle) intersects(c2 Circle) bool {
	dx := c1.X - c2.X
	dy := c1.Y - c2.Y
	distance := math.Sqrt(float64(dx*dx + dy*dy))
	return distance < float64(c1.Radius+c2.Radius)
}

// Генерация списка кругов
func (mg *MapGenerator) generateCircles() ([]Circle, error) {
	rand.Seed(time.Now().UnixNano())
	var circles []Circle
	maxAttempts := 50000

	// Вспомогательная функция для попытки размещения кругов
	tryPlace := func(count, radius int, isSpawn bool) {
		attempts := 0
		for len(circles) < count && attempts < maxAttempts {
			attempts++
			x := rand.Intn(mg.Width*mg.CellSize-2*radius) + radius
			y := rand.Intn(mg.Height*mg.CellSize-2*radius) + radius

			newC := Circle{X: x, Y: y, Radius: radius, IsSpawn: isSpawn}

			canPlace := true
			for _, ex := range circles {
				if newC.intersects(ex) {
					canPlace = false
					break
				}
			}

			if canPlace {
				circles = append(circles, newC)
			}
		}
	}

	// Размещаем spawn круги
	tryPlace(mg.SpawnCount, mg.SpawnRadius, true)

	// Размещаем bedroom круги
	tryPlace(mg.SpawnCount+mg.BedroomCount, mg.BedroomRadius, false)

	if len(circles) < mg.SpawnCount+mg.BedroomCount {
		return nil, fmt.Errorf("невозможно разместить все круги без пересечений")
	}

	return circles, nil
}

// Рисование координатной сетки
func drawGrid(img *image.RGBA, width, height, cellSize int) {
	gridColor := color.RGBA{200, 200, 200, 255}

	for x := 0; x <= width*cellSize; x += cellSize {
		for y := 0; y < height*cellSize; y++ {
			img.Set(x, y, gridColor)
		}
	}

	for y := 0; y <= height*cellSize; y += cellSize {
		for x := 0; x < width*cellSize; x++ {
			img.Set(x, y, gridColor)
		}
	}
}

// Рисование круга
func drawCircle(img *image.RGBA, c Circle, fill, center color.Color) {
	cx, cy, r := c.X, c.Y, c.Radius

	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			if dx*dx+dy*dy <= r*r {
				img.Set(cx+dx, cy+dy, fill)
			}
		}
	}

	img.Set(cx, cy, center)
}

// Генерация изображения карты
func (mg *MapGenerator) generateMapImage() (*image.RGBA, error) {
	circles, err := mg.generateCircles()
	if err != nil {
		return nil, err
	}

	wPx, hPx := mg.Width*mg.CellSize, mg.Height*mg.CellSize
	img := image.NewRGBA(image.Rect(0, 0, wPx, hPx))

	white := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	drawGrid(img, mg.Width, mg.Height, mg.CellSize)

	spawnColor := color.RGBA{255, 0, 0, 255}
	bedroomColor := color.RGBA{0, 0, 255, 255}
	centerColor := color.RGBA{0, 255, 0, 255}

	for _, c := range circles {
		fill := bedroomColor
		if c.IsSpawn {
			fill = spawnColor
		}
		drawCircle(img, c, fill, centerColor)
	}

	return img, nil
}

// HTTP-хендлер
func mapHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	getInt := func(name string) (int, error) {
		v, err := strconv.ParseInt(q.Get(name), 10, 64)
		return int(v), err
	}

	width, err := getInt("width")
	if err != nil || width <= 0 {
		http.Error(w, "Неверный width", http.StatusBadRequest)
		return
	}

	height, err := getInt("height")
	if err != nil || height <= 0 {
		http.Error(w, "Неверный height", http.StatusBadRequest)
		return
	}

	spCnt, err := getInt("spawnscnt")
	if err != nil || spCnt <= 0 {
		http.Error(w, "Неверный spawnscnt", http.StatusBadRequest)
		return
	}

	brCnt, err := getInt("bedroomcnt")
	if err != nil || brCnt <= 0 {
		http.Error(w, "Неверный bedroomcnt", http.StatusBadRequest)
		return
	}

	spRad, err := getInt("spawnradius")
	if err != nil || spRad <= 0 {
		http.Error(w, "Неверный spawnradius", http.StatusBadRequest)
		return
	}

	brRad, err := getInt("bedroomradius")
	if err != nil || brRad <= 0 {
		http.Error(w, "Неверный bedroomradius", http.StatusBadRequest)
		return
	}

	maxGap, err := getInt("maxgap")
	if err != nil || maxGap <= 0 {
		http.Error(w, "Неверный maxgap", http.StatusBadRequest)
		return
	}

	mg := &MapGenerator{
		Width:         width,
		Height:        height,
		SpawnCount:    spCnt,
		BedroomCount:  brCnt,
		SpawnRadius:   spRad,
		BedroomRadius: brRad,
		MaxGap:        maxGap,
		CellSize:      100,
	}

	img, err := mg.generateMapImage()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка генерации: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img)
}

func main() {
	http.HandleFunc("/generate-map", mapHandler)
	fmt.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}
