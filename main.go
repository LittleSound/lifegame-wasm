package main

import (
	"fmt"
	"go-wasm-lifegame/draw"
	"math/rand"
	"syscall/js"
	"time"
)

var global = js.Global()
var document = global.Get("document")

func add(x int, y int) {
	size := 720 / 2
	lives[coordXy(x+size, y+size)] = &Life{
		x:     x + size,
		y:     y + size,
		alive: true,
	}
}

func main() {
	draw.Init()
	rand.Seed(time.Now().UnixNano())
	// initRandomMap(90)
	// add(0, 0)
	// add(0, 1)
	// add(0, 2)
	drawIteration(lives)
	go updateLife(lives, 0)
	js.Global().Set("initMap", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		initMap(args[0].Int())
		return nil
	}))
	showLifeCount()
	onKey()
	onAddPixel()
	select {}
}

var labelIterate = document.Call("getElementById", "iterate")

// 生命数量
func showLifeCount() {
	time.AfterFunc(time.Second/2, func() {
		labelIterate.Set("innerHTML", len(lives))
		showLifeCount()
	})
}

func initMap(size int) {
	initRandomMap(size)
	fmt.Println("棋盘创建完成")
	isUpdateLives = true
	draw.Clear()
	fmt.Println("清屏完成")
	drawIteration(lives)

	fmt.Println("图片绘制完成")
}

// 初始化随机棋盘
func initRandomMap(initSize int) {
	lives = Lives{}
	lifeLe := (initSize * initSize) / 5
	sizeHalf := initSize / 2
	fmt.Println(initSize, lifeLe)
	for i := 0; i < lifeLe; i++ {
		size := 720/2 - sizeHalf
		x := rand.Intn(initSize) + size
		y := rand.Intn(initSize) + size
		lives[coordXy(x, y)] = &Life{
			x:     x,
			y:     y,
			alive: true,
		}
	}
	fmt.Println(len(lives))
}

func addLife(x int, y int) {
	// fmt.Println("addLife", x, y)
	key := coordXy(x, y)
	item := lives[key]
	if item != nil {
		item.alive = true
	} else {
		lives[key] = &Life{
			x:     x,
			y:     y,
			alive: true,
		}
	}
}

// image size (ImageData)
var iw = 720
var ih = 720

// 生死色
const (
	lifeColor = "#33FF33"
	deadColor = "#000000"
)

// Life 生命
type Life struct {
	x      int
	y      int
	alive  bool
	neibor int
}

func (l Life) color() string {
	if l.alive {
		return lifeColor
	}
	return deadColor
}
func (l Lives) getNeibor(x int, y int) {

}

// Lives 很多生命
type Lives map[int]*Life

var lives = Lives{}
var isUpdateLives = false

func iteration(oldLives Lives) Lives {
	newlives := Lives{}
	for key, item := range oldLives {
		// fmt.Println(item.x, item.y, item.alive, item.neibor)
		if item.alive {
			newlives[key] = &Life{
				x:      item.x,
				y:      item.y,
				neibor: updateNeibor(item, oldLives, newlives),
				alive:  item.alive,
			}
			newlives[key].alive = rules(newlives[key])
		}
	}
	return newlives
}

func dateNow() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

var runCount = 0

func updateLife(nowLives Lives, delta time.Duration) {
	time.AfterFunc(time.Second/60-delta, func() {
		start := time.Now().UnixNano()
		// var newlives Lives

		if isUpdateLives {
			isUpdateLives = false
		} else if !pause {
			lives = iteration(nowLives)
		}
		drawIteration(lives)

		// if runCount < 50 {
		// 	runCount++
		// 	updateLife(lives)
		// }
		updateLife(lives, time.Duration(time.Now().UnixNano()-start))
	})
}

func drawIteration(lives Lives) {
	for _, item := range lives {
		draw.AddPixel(item.x, item.y, item.color())
	}
}

var neibors = []struct {
	x int
	y int
}{
	// 1
	{x: -1, y: -1},
	{x: -1, y: 0},
	{x: -1, y: 1},
	// 2
	{x: 0, y: 1},
	{x: 0, y: -1},
	// 3
	{x: 1, y: -1},
	{x: 1, y: 0},
	{x: 1, y: 1},
}

func updateNeibor(item *Life, oldLives Lives, lives Lives) int {
	count := 0
	for _, neibor := range neibors {
		x := item.x + neibor.x
		y := item.y + neibor.y
		key := coordXy(x, y)
		if oldLives[key] != nil && oldLives[key].alive {
			count++
			continue
		}
		if lives[key] != nil {
			lives[key].neibor++
			lives[key].alive = rules(lives[key])
		} else {
			lives[key] = new(Life)
			lives[key].neibor++
			lives[key].x = x
			lives[key].y = y
		}
	}
	return count
}

// 规则
func rules(item *Life) bool {
	if item.neibor == 3 {
		return true
	}
	if item.neibor == 2 {
		return item.alive
	}
	return false
}

// coordXy
func coordXy(x int, y int) int {
	return y*iw + x
}

// 暂停
var pause = false

// 按键监听
func onKey() {
	document.Set("onkeyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		evt := args[0]
		currKey := evt.Get("keyCode").Int()
		if currKey == 0 {
			currKey = evt.Get("which").Int()
		}
		if currKey == 0 {
			currKey = evt.Get("charCode").Int()
		}
		if currKey == 32 {
			pause = !pause
		}

		return nil
	}))
}

func onAddPixel() {
	draw.OnAddPixel = func(x, y int) {
		addLife(x, y)
	}
}
