package draw

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"syscall/js"
	"time"

	"github.com/llgcode/draw2d/draw2dimg"
)

// viewport size (canvas)
var vw = 500
var vh = 500

// image size (ImageData)
var iw = 720
var ih = 720

// 背景色
var bgColor = color.Black

var global = js.Global()
var document = global.Get("document")

var (
	scale   = float64(1)
	cameraX = float64(iw / 2)
	cameraY = float64(ih / 2)
)

func dateNow() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// Init 初始化
func Init() {
	// initCamera()
	initColorInput()
	initCanvas()

	initMouseEvents()
	updateCamera()
	initResize()

	document.Call("addEventListener", "wheel", js.FuncOf(onWheel), false)

	go updateFrame(time.Duration(0))
}

// to handle the camera we use a DOMMatrix object
// which offers a few handful methods
var camera = global.Get("DOMMatrix").New()

// var arr = document.Call("querySelectorAll", "[type='range']")
// var (
// 	zInput = arr.Index(0)
// 	xInput = arr.Index(1)
// 	yInput = arr.Index(2)
// )

// // 初始化相机
// func initCamera() {
// 	for _, v := range []js.Value{zInput, xInput, yInput} {
// 		v.Set("oninput", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
// 			updateCamera()
// 			return nil
// 		}))
// 	}
// }

var labelX = document.Call("getElementById", "x")
var labelY = document.Call("getElementById", "y")
var labelSize = document.Call("getElementById", "size")

// 更新相机
func updateCamera() {
	// z, _ := strconv.ParseFloat(zInput.Get("value").String(), 64)
	// x, _ := strconv.ParseFloat(xInput.Get("value").String(), 64)
	// y, _ := strconv.ParseFloat(yInput.Get("value").String(), 64)
	z := scale
	x := cameraX
	y := cameraY

	camera.Set("d", z)
	camera.Set("a", z)
	camera.Set("e", float64(vw)/2-x*z)
	camera.Set("f", float64(vh)/2-y*z)

	labelX.Set("innerHTML", math.Round(x))
	labelY.Set("innerHTML", math.Round(y))
	labelSize.Set("innerHTML", math.Round(z))
}

var colorinput = document.Call("querySelector", "input[type='color']")
var colorStr = "#FF0000"

// 初始化颜色输入框
func initColorInput() {
	colorinput.Set("value", colorStr)
	colorinput.Set("oninput", js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
		colorStr = colorinput.Get("value").String()
		return nil
	}))
}

// the visible canvas
var canvas = document.Call("querySelector", "canvas")
var ctx = canvas.Call("getContext", "2d")

// we hold our pixel's data directly in an ImageData
var imgData = global.Get("ImageData").New(iw, ih)

// 使用 image 包绘制图形
var img = image.NewRGBA(image.Rect(0, 0, iw, ih))
var graphicCtx = draw2dimg.NewGraphicContext(img)
var imglen = len(img.Pix)

// 复制缓冲区
var copyBuffer = js.Global().Get("Uint8Array").New(len(img.Pix))

// use a 32 bit view to access each pixel directly as a single value
// var pixels = global.Get("Uint32Array").New(len(img.Pix))

// an other canvas, kept off-screen
var scaler = document.Call("createElement", "canvas")
var scalerctx = scaler.Call("getContext", "2d")

// OnAddPixel 添加像素回掉
var OnAddPixel func(x int, y int)

// 初始化画布
func initCanvas() {
	graphicCtx.SetFillColor(bgColor)
	graphicCtx.MoveTo(0, 0)
	graphicCtx.LineTo(float64(iw), 0)
	graphicCtx.LineTo(float64(iw), float64(ih))
	graphicCtx.LineTo(0, float64(ih))
	graphicCtx.Close()
	graphicCtx.Fill()

	canvas.Set("width", vw)
	canvas.Set("height", vh)
	scaler.Set("width", iw)
	scaler.Set("height", ih)
}

// 鼠标
var mouse struct {
	x     int
	y     int
	ldown bool
	rdown bool
}

// 鼠标移动事件
func canvasOnmousemove(_ js.Value, args []js.Value) interface{} {
	evt := args[0]
	canvasBBox := canvas.Call("getBoundingClientRect")
	// relative to the canvas viewport
	x := evt.Get("clientX").Int() - canvasBBox.Get("left").Int()
	y := evt.Get("clientY").Int() - canvasBBox.Get("top").Int()
	// transform it by the current camera
	point := camera.Call("inverse").Call("transformPoint", map[string]interface{}{"x": x, "y": y})
	mouse.x = point.Get("x").Int()
	mouse.y = point.Get("y").Int()

	if mouse.ldown && OnAddPixel != nil {
		// AddPixel(mouse.x, mouse.y, colorStr)
		OnAddPixel(mouse.x, mouse.y)
	}
	if mouse.rdown {
		cameraX += -evt.Get("movementX").Float() / scale
		cameraY += -evt.Get("movementY").Float() / scale
		updateCamera()
	}
	return nil
}

// 初始化鼠标事件
func initMouseEvents() {
	canvas.Set("onmousemove", js.FuncOf(canvasOnmousemove))
	canvas.Set("onmousedown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		button := args[0].Get("button")
		fmt.Println("button ⬇️", button)
		if button.Int() == 2 {
			mouse.rdown = true
		} else {
			mouse.ldown = true
		}
		return nil
	}))
	canvas.Set("onmouseup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		button := args[0].Get("button")
		fmt.Println("button ⬆️", button)
		if button.Int() == 2 {
			mouse.rdown = false
		} else {
			mouse.ldown = false
		}
		return nil
	}))
}

// 绘制
func draw() {
	js.CopyBytesToJS(copyBuffer, img.Pix)
	imgData.Get("data").Call("set", copyBuffer)

	// first draw the ImageData on the scaler canvas
	scalerctx.Call("putImageData", imgData, 0, 0)

	// reset the transform to default
	ctx.Call("setTransform", 1, 0, 0, 1, 0, 0)
	ctx.Call("clearRect", 0, 0, vw, vh)
	// set the transform to the camera
	ctx.Call("setTransform", camera)

	// pixel art so no antialising
	ctx.Set("imageSmoothingEnabled", false)
	// draw the image data, scaled on the visible canvas
	ctx.Call("drawImage", scaler, 0, 0)
	// draw the (temp) cursor
	ctx.Set("fillStyle", colorStr)
	ctx.Call("fillRect", mouse.x, mouse.y, 1, 1)
}

// 转化 hex 色值到十六进制计数法
// func parseColor(str string) string {
// 	splited := strings.Split(str, "")[1:]
// 	hex := make([]string, 3)
// 	for i := 0; i < len(splited); i += 2 {
// 		hex = append(hex, splited[i]+splited[i+1])
// 	}
// 	for i, j := 0, len(hex)-1; i < j; i, j = i+1, j-1 {
// 		hex[i], hex[j] = hex[j], hex[i]
// 	}
// 	return "0xFF" + strings.Join(hex, "")
// }

// AddPixel 添加像素
func AddPixel(x int, y int, colorStr string) {
	index := y*iw + x
	if index > 0 && index < imglen {
		colorHex, err := parseHexColor(colorStr)
		if err != nil {
			fmt.Println(err)
		}
		img.Set(x, y, colorHex)
	}
}

// Clear 清屏
func Clear() {
	graphicCtx.SetFillColor(bgColor)
	graphicCtx.MoveTo(0, 0)
	graphicCtx.LineTo(float64(iw), 0)
	graphicCtx.LineTo(float64(iw), float64(ih))
	graphicCtx.LineTo(0, float64(ih))
	graphicCtx.Close()
	graphicCtx.Fill()
}

// 转换 hex 色值为 color.RGBA 结构类型
func parseHexColor(s string) (c color.RGBA, err error) {
	if s == "#33FF33" {
		return color.RGBA{
			R: 0x33,
			G: 0xFF,
			B: 0x33,
			A: 0xFF,
		}, nil
	}
	return color.RGBA{
		A: 0xFF,
	}, nil
	// c.A = 0xff
	// switch len(s) {
	// case 7:
	// 	_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	// case 4:
	// 	_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
	// 	// Double the hex digits:
	// 	c.R *= 17
	// 	c.G *= 17
	// 	c.B *= 17
	// default:
	// 	err = fmt.Errorf("invalid length, must be 7 or 4")
	// }
	// return
}

// 窗口大小变更事件回调
func resizeCanvas() {
	vw = document.Get("body").Call("getBoundingClientRect").Get("width").Int()
	vh = document.Get("body").Call("getBoundingClientRect").Get("height").Int()

	canvas.Set("width", vw)
	canvas.Set("height", vh)

	updateCamera()
}

// 初始化窗口大小变更事件
func initResize() {
	global.Get("window").Call("addEventListener", "resize", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resizeCanvas()
		return nil
	}), false)
	resizeCanvas()
}

// 鼠标滚轮事件回调函数
func onWheel(_ js.Value, args []js.Value) interface{} {
	e := args[0]
	scale += e.Get("deltaY").Float() * 0.1
	if scale < 1 {
		scale = 1
	} else if scale > 20 {
		scale = 20
	}
	updateCamera()
	return nil
}

func updateFrame(delta time.Duration) {
	time.AfterFunc(time.Second/60-delta, func() {
		startTime := time.Now().UnixNano()
		draw()
		deltaTime := time.Now().UnixNano() - startTime
		updateFrame(time.Duration(deltaTime))
	})
}
