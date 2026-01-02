package main

import (
	"image/color"
	"machine"
	"time"
	"unicode"

	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7789"
)

// 短い型名エイリアス
type Image = pixel.Image[pixel.RGB565BE]

const (
	// ドット絵サイズ
	PAT_W      = 5
	PAT_H      = 5
	BLOCK_SIZE = 12 // 1ドットを何ピクセルで拡大して描画するか（小さめに変更）
)

func main() {
	display := initDisplay()

	pattern, pal := createPatternAndPalette()
	img, bgImg, w, h := createImages(pattern, pal)

	leftBtn, upBtn, rightBtn, downBtn := initButtons()

	// 初期位置（中央）
	posX := (240 - w) / 2
	posY := (240 - h) / 2

	// 初回描画
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	display.DrawBitmap(int16(posX), int16(posY), *img)

	// 会話ウィンドウ用ゴルーチン（1分ごとに挨拶）を別関数で起動
	go startSpeech(display, 8, 200, "Hello!", time.Minute)

	// ブザー初期化と音楽ゴルーチン（非同期で1フレーズ約30秒）を起動
	buzzer := machine.D3
	buzzer.Configure(machine.PinConfig{Mode: machine.PinOutput})
	go startMusic(buzzer)

	// メインループ
	runLoop(display, img, bgImg, w, h, posX, posY, leftBtn, upBtn, rightBtn, downBtn, buzzer)
}


// 初期化: SPI とディスプレイ
func initDisplay() st7789.Device {
	machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 16000000,
		Mode:      0,
	})
	display := st7789.New(machine.SPI1,
		machine.GPIO9,
		machine.GPIO12,
		machine.GPIO13,
		machine.GPIO14)
	display.Configure(st7789.Config{
		Height:   240,
		Width:    240,
	})
	return display
}

// パターンとパレットを返す
func createPatternAndPalette() ([][]int, map[int][3]uint8) {
	pattern := [][]int{
		{0, 1, 0, 1, 0},
		{1, 2, 1, 2, 1},
		{1, 3, 1, 3, 1},
		{1, 1, 3, 1, 1},
		{0, 1, 2, 1, 0},
	}
	pal := map[int][3]uint8{
		0: {0, 0, 0},
		1: {120, 170, 200},
		2: {255, 255, 255},
		3: {0, 0, 0},
		4: {160, 80, 40},
		5: {255, 0, 0},
	}
	return pattern, pal
}

// 画像バッファを生成
func createImages(pattern [][]int, pal map[int][3]uint8) (*Image, *Image, int, int) {
	w := PAT_W * BLOCK_SIZE
	h := PAT_H * BLOCK_SIZE
	img := pixel.NewImage[pixel.RGB565BE](w, h)
	bgImg := pixel.NewImage[pixel.RGB565BE](w, h)
	black := pixel.NewColor[pixel.RGB565BE](0, 0, 0)
	for i := 0; i < w*h; i++ {
		x := i % w
		y := i / w
		bgImg.Set(x, y, black)
	}
	for i := 0; i < w*h; i++ {
		x := i % w
		y := i / w
		px := x / BLOCK_SIZE
		py := y / BLOCK_SIZE
		v := pattern[py][px]
		c := pal[v]
		pc := pixel.NewColor[pixel.RGB565BE](c[0], c[1], c[2])
		img.Set(x, y, pc)
	}
	return &img, &bgImg, w, h
}

// ボタン初期化
func initButtons() (machine.Pin, machine.Pin, machine.Pin, machine.Pin) {
	leftBtn := machine.D16
	upBtn := machine.D5
	rightBtn := machine.D28
	downBtn := machine.D22
	leftBtn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	upBtn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	rightBtn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	downBtn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return leftBtn, upBtn, rightBtn, downBtn
}

// メインループ（描画と入力処理）
func runLoop(display st7789.Device, img, bgImg *Image, w, h int, posX, posY int, leftBtn, upBtn, rightBtn, downBtn, buzzer machine.Pin) {
	// 前の描画位置（消去に使用）
	prevX := posX
	prevY := posY

	// 前回状態（エッジ検出）
	prevL, prevU, prevR, prevD := true, true, true, true


	for {
		curL := leftBtn.Get()
		curU := upBtn.Get()
		curR := rightBtn.Get()
		curD := downBtn.Get()

		moved := false
		if !curL && prevL {
			posX -= BLOCK_SIZE
			moved = true
		}
		if !curR && prevR {
			posX += BLOCK_SIZE
			moved = true
		}
		if !curU && prevU {
			posY -= BLOCK_SIZE
			moved = true
		}
		if !curD && prevD {
			posY += BLOCK_SIZE
			moved = true
		}

		// 範囲制限
		if posX < 0 {
			posX = 0
		}
		if posY < 0 {
			posY = 0
		}
		if posX > 240-w {
			posX = 240 - w
		}
		if posY > 240-h {
			posY = 240 - h
		}

		if moved {
			// 簡易クリック音
			playTone(buzzer, 880, 60*time.Millisecond)

			// 前位置だけを黒で消して、新位置に描画 -> 全面クリアを避けてちらつきを抑える
			display.DrawBitmap(int16(prevX), int16(prevY), *bgImg)
			display.DrawBitmap(int16(posX), int16(posY), *img)

			// 前位置を更新
			prevX = posX
			prevY = posY
		}
		prevL, prevU, prevR, prevD = curL, curU, curR, curD
		time.Sleep(30 * time.Millisecond)


	}

}

// 簡易 5x7 ASCII フォント（使用する文字のみ）
var ascii5x7 = map[rune][7]uint8{
	'A': {0x1E, 0x05, 0x05, 0x1E, 0x00, 0x00, 0x00},
	'C': {0x1E, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00},
	'H': {0x11, 0x11, 0x1F, 0x11, 0x11, 0x00, 0x00},
	'I': {0x1F, 0x04, 0x04, 0x04, 0x1F, 0x00, 0x00},
	'K': {0x11, 0x12, 0x1C, 0x12, 0x11, 0x00, 0x00},
	'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x00, 0x00},
	'O': {0x0E, 0x11, 0x11, 0x11, 0x0E, 0x00, 0x00},
	'W': {0x11, 0x11, 0x15, 0x0A, 0x11, 0x00, 0x00},
	' ': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	'E': {0x1F, 0x10, 0x1E, 0x10, 0x1F, 0x00, 0x00},
	'L': {0x10, 0x10, 0x10, 0x10, 0x1F, 0x00, 0x00},
	'!': {0x04, 0x04, 0x04, 0x04, 0x00, 0x04, 0x00},
}

// 指定位置に幅/高さで黒矩形を描く（消去に利用）
func clearRect(display st7789.Device, x, y, width, height int) {
	// 幅×高さの黒イメージを作って一回だけ描画する（効率化）
	if width <= 0 || height <= 0 {
		return
	}
	img := pixel.NewImage[pixel.RGB565BE](width, height)
	black := pixel.NewColor[pixel.RGB565BE](0, 0, 0)
	for i := 0; i < width*height; i++ {
		tx := i % width
		ty := i / width
		img.Set(tx, ty, black)
	}
	display.DrawBitmap(int16(x), int16(y), img)
}

// テキストイメージを作成して描画する（簡易）
func showSpeech(display st7789.Device, x, y int, text string) {
	scale := 2
	fw := 5
	fh := 7
	tw := (fw+1)*len(text) * scale
	th := fh * scale
	txt := pixel.NewImage[pixel.RGB565BE](tw, th)
	fg := pixel.NewColor[pixel.RGB565BE](255, 255, 255)
	// 背景（少し明るめの枠）
	bg := pixel.NewColor[pixel.RGB565BE](40, 40, 80)
	for i := 0; i < tw*th; i++ {
		tx := i % tw
		ty := i / tw
		txt.Set(tx, ty, bg)
	}
	// 文字描画（単一ループで処理）
	runes := []rune(text)
	charWidth := (fw + 1) * scale
	for i := 0; i < tw*th; i++ {
		tx := i % tw
		ty := i / tw
		ci := tx / charWidth
		if ci < 0 || ci >= len(runes) {
			txt.Set(tx, ty, bg)
			continue
		}
		withinX := tx - ci*charWidth
		if withinX >= fw*scale || ty >= fh*scale {
			txt.Set(tx, ty, bg)
			continue
		}
		col := withinX / scale
		row := ty / scale
		ch := unicode.ToUpper(runes[ci])
		pattern, ok := ascii5x7[ch]
		if !ok {
			pattern = ascii5x7[' ']
		}
		bits := pattern[row]
		if (bits>>(4-col))&1 == 1 {
			txt.Set(tx, ty, fg)
		} else {
			txt.Set(tx, ty, bg)
		}
	}
	// バブル枠（白線）を1x1で作り描画
	border := pixel.NewColor[pixel.RGB565BE](200, 200, 255)
	small := pixel.NewImage[pixel.RGB565BE](1, 1)
	small.Set(0, 0, border)
	for bx := -2; bx < tw+2; bx++ {
		display.DrawBitmap(int16(x+bx), int16(y-2), small)
	}
	// 描画
	display.DrawBitmap(int16(x), int16(y), txt)
}

// startSpeech は会話ウィンドウを定期的に表示するゴルーチン用関数です。
func startSpeech(display st7789.Device, sx, sy int, msg string, interval time.Duration) {
	for {
		showSpeech(display, sx, sy, msg)
		// 表示3秒
		time.Sleep(10 * time.Second)
		// 消去
		clearRect(display, sx-4, sy-4, 120, 28)
		// 残りを待つ（interval は合計の周期）
		wait := interval - 3*time.Second
		if wait < 0 {
			wait = 0
		}
		time.Sleep(wait)
	}
}

// startMusic は非同期で1フレーズ（約30秒）ほどの楽しいメロディを鳴らします。
func startMusic(pin machine.Pin) {
	type note struct {
		freq uint16
		dur  time.Duration
	}
	// 明るい短いフレーズ（Ode-like シンプルメロディ）
	melody := []note{
		{330, 300 * time.Millisecond}, // E4
		{330, 300 * time.Millisecond}, // E4
		{349, 600 * time.Millisecond}, // F4
		{392, 600 * time.Millisecond}, // G4

		{392, 300 * time.Millisecond}, // G4
		{349, 300 * time.Millisecond}, // F4
		{330, 600 * time.Millisecond}, // E4
		{294, 600 * time.Millisecond}, // D4

		{261, 600 * time.Millisecond}, // C4
		{261, 600 * time.Millisecond}, // C4
		{294, 300 * time.Millisecond}, // D4
		{330, 600 * time.Millisecond}, // E4

		{330, 300 * time.Millisecond}, // E4
		{294, 900 * time.Millisecond}, // D4 (long)
	}

	for {
		start := time.Now()
		idx := 0
		// 単一ループで時間を見つつメロディの各音を順に鳴らす
		for time.Since(start) < 30*time.Second {
			n := melody[idx%len(melody)]
			playTone(pin, n.freq, n.dur)
			// 音符間の短いスペース
			time.Sleep(80 * time.Millisecond)
			idx++
		}
		// 少し休止してから次のフレーズ
		time.Sleep(1 * time.Second)
	}
}

// シンプルなトーン生成（ブロック的）
func playTone(pin machine.Pin, freq uint16, duration time.Duration) {
	if freq == 0 {
		time.Sleep(duration)
		return
	}
	period := time.Second / time.Duration(freq)
	half := period / 2
	cycles := int(duration / period)
	if cycles < 1 {
		cycles = 1
	}
	for i := 0; i < cycles; i++ {
		pin.High()
		time.Sleep(half)
		pin.Low()
		time.Sleep(half)
	}
}
