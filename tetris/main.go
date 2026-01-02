package main

import (
	"machine"
	"math/rand"
	"time"

	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7789"
)

type Image = pixel.Image[pixel.RGB565BE]

const (
	GRID_W = 10
	GRID_H = 20
	BLOCK_SIZE = 12
)

// Board: 0 empty, >0 color index
var board [GRID_W * GRID_H]uint8

// Tetromino definitions (rotations as slices of 4 (x,y) pairs)
var pieces = [][][]int{
	// I
	{{0,1, 1,1, 2,1, 3,1},{2,0,2,1,2,2,2,3}},
	// O
	{{1,0,2,0,1,1,2,1}},
	// T
	{{1,0,0,1,1,1,2,1},{1,0,1,1,2,1,1,2},{0,1,1,1,2,1,1,2},{1,0,0,1,1,1,1,2}},
	// S
	{{1,0,2,0,0,1,1,1},{1,0,1,1,2,1,2,2}},
	// Z
	{{0,0,1,0,1,1,2,1},{2,0,1,1,2,1,1,2}},
	// J
	{{0,0,0,1,1,1,2,1},{1,0,2,0,1,1,1,2},{0,1,1,1,2,1,2,2},{1,0,1,1,0,2,1,2}},
	// L
	{{2,0,0,1,1,1,2,1},{1,0,1,1,1,2,2,2},{0,1,1,1,2,1,0,2},{0,0,1,0,1,1,1,2}},
}

var colors = []pixel.RGB565BE{
	pixel.NewColor[pixel.RGB565BE](0,0,0),
	pixel.NewColor[pixel.RGB565BE](200,80,80),
	pixel.NewColor[pixel.RGB565BE](80,200,120),
	pixel.NewColor[pixel.RGB565BE](120,120,200),
	pixel.NewColor[pixel.RGB565BE](200,200,80),
	pixel.NewColor[pixel.RGB565BE](200,120,200),
	pixel.NewColor[pixel.RGB565BE](120,200,200),
}

// current piece state
type Piece struct {
	kind int
trot int
	x int
	y int
}

var cur Piece

/**
main はゲームの初期化とメインループを実行します。
Parameters: None
Return: None
*/
func main() {
	rand.Seed(time.Now().UnixNano())
	display := initDisplay()
	left, up, right, down := initButtons()
	buzzer := machine.D3
	buzzer.Configure(machine.PinConfig{Mode: machine.PinOutput})
	// 非同期 BGM を開始（パブリックドメインのメロディ）
	go startBGM(buzzer)

	clearBoard()
	spawnPiece()

	// initial draw
	drawBoard(display)

	// main loop: input and gravity
	lastFall := time.Now()
	fallInterval := 700 * time.Millisecond
	for {
		// read buttons (no deep nesting)
		if !left.Get() { tryMove(-1,0) }
		if !right.Get() { tryMove(1,0) }
		if !up.Get() { rotatePiece() }
		if !down.Get() { // soft drop
			if tryMove(0,1) { playTone(buzzer,880,40*time.Millisecond) }
		}

		// gravity
		if time.Since(lastFall) >= fallInterval {
			if !tryMove(0,1) {
				lockPiece()
				clearLines()
				spawnPiece()
			}
			lastFall = time.Now()
			drawBoard(display)
		}

		// small sleep to debounce
		time.Sleep(80 * time.Millisecond)
	}
}

/**
initDisplay は SPI と ST7789 ディスプレイを初期化してデバイスを返します。
Parameters: None
Return: 初期化済みの `st7789.Device`
*/
func initDisplay() st7789.Device {
	machine.SPI1.Configure(machine.SPIConfig{Frequency:16000000, Mode:0})
	d := st7789.New(machine.SPI1, machine.GPIO9, machine.GPIO12, machine.GPIO13, machine.GPIO14)
	d.Configure(st7789.Config{Height:240, Width:240})
	return d
}

/**
initButtons は左右上下のボタンピンを設定して返します。
Parameters: None
Return: left, up, right, down の各 `machine.Pin`
*/
func initButtons() (machine.Pin, machine.Pin, machine.Pin, machine.Pin) {
	l := machine.D16
	u := machine.D5
	r := machine.D28
	d := machine.D22
	l.Configure(machine.PinConfig{Mode:machine.PinInputPullup})
	u.Configure(machine.PinConfig{Mode:machine.PinInputPullup})
	r.Configure(machine.PinConfig{Mode:machine.PinInputPullup})
	d.Configure(machine.PinConfig{Mode:machine.PinInputPullup})
	return l,u,r,d
}

// initButtons は左右上下のボタンピンを設定して返します。
// Parameters: None
// Return: left, up, right, down の各 `machine.Pin`

/**
clearBoard は内部ボードをすべて 0（空）にリセットします。
Parameters: None
Return: None
*/
func clearBoard() {
	for i:=0;i<GRID_W*GRID_H;i++{ board[i]=0 }
}

/**
spawnPiece はランダムなテトリミノを生成して `cur` にセットします。
もしスポーン位置が衝突している場合はボードをクリアします。
Parameters: None
Return: None
*/
func spawnPiece() {
	k := rand.Intn(len(pieces))
	cur = Piece{kind:k, trot:0, x:GRID_W/2 - 2, y:0}
	// if spawn collides -> reset board
	if collides(cur.x, cur.y, cur.trot) {
		clearBoard()
	}
}

/**
collides は指定位置・回転で現在のピースがボードと衝突するか判定します。
Parameters:
 - px, py: ピースの左上基準の座標
 - rot: ピースの回転インデックス
Return: 衝突する場合は true、しない場合は false
*/
func collides(px, py, rot int) bool {
	shape := pieces[cur.kind][rot%len(pieces[cur.kind])]
	for i:=0;i<8;i+=2{
		x := px + shape[i]
		y := py + shape[i+1]
		if x < 0 || x >= GRID_W || y < 0 || y >= GRID_H { return true }
		if board[y*GRID_W + x] != 0 { return true }
	}
	return false
}

/**
tryMove は現在ピースを (dx,dy) 移動できるか試み、可能なら更新します。
Parameters:
 - dx, dy: 移動量
Return: 移動に成功したら true
*/
func tryMove(dx, dy int) bool {
	nx := cur.x + dx
	ny := cur.y + dy
	if !collides(nx, ny, cur.trot) {
		cur.x = nx
		cur.y = ny
		return true
	}
	return false
}

/**
rotatePiece は現在ピースを時計回りに回転させます（衝突検査あり）。
Parameters: None
Return: None
*/
func rotatePiece() {
	newRot := (cur.trot + 1) % len(pieces[cur.kind])
	if !collides(cur.x, cur.y, newRot) {
		cur.trot = newRot
	}
}

/**
lockPiece は現在ピースをボードに固定（書き込み）します。
Parameters: None
Return: None
*/
func lockPiece() {
	shape := pieces[cur.kind][cur.trot%len(pieces[cur.kind])]
	for i:=0;i<8;i+=2{
		x := cur.x + shape[i]
		y := cur.y + shape[i+1]
		if x>=0 && x<GRID_W && y>=0 && y<GRID_H {
			board[y*GRID_W + x] = uint8((cur.kind % (len(colors)-1)) + 1)
		}
	}
}

/**
clearLines は揃った行を検出して削除し、上の行を下に詰めます。
Parameters: None
Return: None
*/
func clearLines() {
	// check each row; avoid nested column loops by scanning linear section per row
	write := 0
	for row:=0; row<GRID_H; row++ {
		full := true
		base := row*GRID_W
		for c:=0;c<GRID_W;c++ { if board[base+c]==0 { full=false; break } }
		if !full {
			// copy row to write position if different
			if write != row {
				for c:=0;c<GRID_W;c++ { board[write*GRID_W + c] = board[row*GRID_W + c] }
			}
			write++
		}
	}
	// clear remaining rows
	for r:=write; r<GRID_H; r++ { for c:=0;c<GRID_W;c++ { board[r*GRID_W + c]=0 } }
}

/**
drawBoard は内部ボードと現在ピースを `display` に描画します。
Parameters:
 - display: 描画先の `st7789.Device`
Return: None
*/
func drawBoard(display st7789.Device) {
	w := GRID_W * BLOCK_SIZE
	h := GRID_H * BLOCK_SIZE
	img := pixel.NewImage[pixel.RGB565BE](w,h)
	bg := pixel.NewColor[pixel.RGB565BE](0,0,20)
	// fill background single loop
	for i:=0;i<w*h;i++{ x := i % w; y := i / w; img.Set(x,y,bg) }
	// draw board cells single loop
	for i:=0;i<GRID_W*GRID_H;i++{
		v := board[i]
		if v==0 { continue }
		x := (i % GRID_W) * BLOCK_SIZE
		y := (i / GRID_W) * BLOCK_SIZE
		fillBlock(img, x, y, colors[v])
	}
	// draw current piece
	shape := pieces[cur.kind][cur.trot%len(pieces[cur.kind])]
	for i:=0;i<8;i+=2{
		x := (cur.x + shape[i]) * BLOCK_SIZE
		y := (cur.y + shape[i+1]) * BLOCK_SIZE
		fillBlock(img, x, y, colors[(cur.kind%(len(colors)-1))+1])
	}
	// draw to display
	display.DrawBitmap(int16((240-w)/2), int16((240-h)/2), img)
}

// drawBoard は内部ボードと現在ピースを `display` に描画します。
// Parameters:
//  - display: 描画先の `st7789.Device`
// Return: None

/**
fillBlock はイメージ `img` の (ox,oy) から BLOCK_SIZE の正方形を色 `col` で塗りつぶします。
Parameters:
 - img: 描画対象のイメージ
 - ox, oy: 塗り始めの左上ピクセル座標
 - col: 塗りつぶす色（`pixel.RGB565BE`）
Return: None
*/
func fillBlock(img Image, ox, oy int, col pixel.RGB565BE) {
	// fill BLOCK_SIZE x BLOCK_SIZE with color using single loop
	for i:=0;i<BLOCK_SIZE*BLOCK_SIZE;i++{
		x := ox + (i % BLOCK_SIZE)
		y := oy + (i / BLOCK_SIZE)
		img.Set(x,y,col)
	}
}

// simple tone function reused
/**
playTone は指定ピンに矩形波を出力して単純なトーンを鳴らします。
Parameters:
 - pin: 出力する `machine.Pin`
 - freq: 周波数（Hz）
 - duration: 再生時間
Return: None
*/
func playTone(pin machine.Pin, freq uint16, duration time.Duration) {
	if freq == 0 { time.Sleep(duration); return }
	period := time.Second / time.Duration(freq)
	half := period / 2
	cycles := int(duration / period)
	if cycles < 1 { cycles = 1 }
	for i:=0;i<cycles;i++{ pin.High(); time.Sleep(half); pin.Low(); time.Sleep(half) }
}

/**
startBGM は著作権フリー（パブリックドメイン）な曲（Twinkle Twinkle）を
非同期でループ再生します。
Parameters:
 - pin: 出力する `machine.Pin`
Return: なし
*/
func startBGM(pin machine.Pin) {
	type note struct { freq uint16; dur time.Duration }
	// Twinkle Twinkle の簡易メロディ（パブリックドメイン）
	melody := []note{
		{261, 300}, {261,300}, {392,300}, {392,300}, {440,300}, {440,300}, {392,600},
		{349,300}, {349,300}, {330,300}, {330,300}, {294,300}, {294,300}, {261,600},
		{392,300}, {392,300}, {349,300}, {349,300}, {330,300}, {330,300}, {294,600},
		{392,300}, {392,300}, {349,300}, {349,300}, {330,300}, {330,300}, {294,600},
	}

	for {
		start := time.Now()
		idx := 0
		// 約30秒のフレーズを単一ループで再生
		for time.Since(start) < 30*time.Second {
			n := melody[idx%len(melody)]
			playTone(pin, n.freq, n.dur*time.Millisecond)
			time.Sleep(80 * time.Millisecond)
			idx++
		}
		time.Sleep(2 * time.Second)
	}
}
