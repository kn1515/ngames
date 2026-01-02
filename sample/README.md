**Project**: TinyGo ST7789 Sample

- **Description**: 簡易ドット絵と会話メッセージを表示する TinyGo サンプル。方向ボタンでキャラクターを移動し、1 分ごとに挨拶を表示します。

**Required Parts**

- **Microcontroller**: Raspberry Pi Pico / RP2040 ボード（Waveshare RP2040 Zero 等）
- **Display**: ST7789 240x240 SPI TFT モジュール
- **Buttons**: 押しボタンスイッチ ×4（左右上下）
- **Buzzer**: 小型アクティブ/パッシブブザー（任意）
- **Wires & Breadboard**: ジャンパーワイヤとブレッドボード
- **USB ケーブル**: ボード給電および書き込み用

**Pin / Wiring (コードに合わせた接続)**

- **ST7789 (SPI)**:

  - **SCLK (CLK)**: ボードの SPI1 SCLK
  - **MOSI (DIN)**: ボードの SPI1 MOSI
  - **DC**: GPIO9
  - **RST**: GPIO12
  - **CS**: GPIO13
  - **BL (バックライト)**: GPIO14
  - **VCC**: 3.3V
  - **GND**: GND

  注: モジュール側の信号名はメーカー毎に異なります。上の GPIO 番号はソース内の `st7789.New(machine.SPI1, machine.GPIO9, machine.GPIO12, machine.GPIO13, machine.GPIO14)` に対応しています。

- **ボタン（入力）**:

  - 左ボタン: `D16`
  - 上ボタン: `D5`
  - 右ボタン: `D28`
  - 下ボタン: `D22`
  - 配線: ボタンの片側をそれぞれの GPIO に、もう片側を GND に接続（コードは内部プルアップを使用しています）

- **ブザー**:
  - 出力: `D3`
  - 接続: ブザーの + を `D3`、- を GND（パッシブブザーを使う場合は適切なドライブ回路に注意）

**ソフトウェア前提**

- **TinyGo** がインストール済みで、`tinygo` コマンドが使えること
- 必要なドライバ: `tinygo.org/x/drivers/pixel` と `tinygo.org/x/drivers/st7789`（ソースは go.mod を通して取得されます）

**ビルド & フラッシュ (Windows PowerShell の例)**

- サンプルディレクトリに移動して次を実行します:

```powershell
tinygo flash --target waveshare-rp2040-zero --size short main.go
```

- ボードが USB 接続されていることを確認してください。ターゲットやサイズは使用ボードに合わせて調整してください。

**操作方法**

- **方向ボタン**: キャラクターを 1 ドット（`BLOCK_SIZE` 分のピクセル）ずつ移動します。
- **クリック音**: 移動時にブザーが短く鳴ります（`playTone` 関数）。
- **会話バブル**: 画面下部に 1 分ごとに "Hello!" が 3 秒間表示されます。表示後は枠が消去されます。

**カスタマイズ**

- 文字列/周期の変更: `main.go` の `startSpeech(display, 8, 200, "Hello!", time.Minute)` の引数を変更してください。
- ドット絵、パレットの変更: `createPatternAndPalette()` の `pattern` / `pal` を編集してください。
- フォント追加: `ascii5x7` にルーンパターンを追加することで表示できる文字を増やせます。

**トラブルシューティング**

- 画面が真っ黒: 配線（VCC/GND/CS/DC/RST/BL）と SPI ピンの接続を確認。
- ボタンが効かない: 各ボタンが GND に正しく接続されているか、また `initButtons()` のピン割り当てが実機のピン配置に合っているか確認。
- ビルドエラー: TinyGo のバージョンとターゲットがサポート対象か確認してください。

**License / Notes**

- このサンプルは教育目的のシンプルなデモです。実運用では入力デバウンスや電源管理、バックライト制御、安全なブザー駆動回路などを追加してください。

---
