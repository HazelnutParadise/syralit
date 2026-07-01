# Syralit 規格書

**狀態：** Draft（API 穩定前皆可破壞性調整）
**定位：** Go-native、Streamlit 風格的互動式資料 App 框架
**核心理念：** 用很少的 Go 程式碼寫出互動式資料 App，並讓 Insyra 成為官方一等資料引擎。

> 本規格不綁定具體版本號。所有交付節奏以 **MVP / Next / Later** 三個階段描述（見 §20）。真正的 semver 與相容性承諾，等到第一個可發佈版本成形時再另行定義。

---

## 1. 專案定位

Syralit 是一個用 Go 開發的互動式資料 App 框架，使用體驗接近 Streamlit，但遵守 Go 的語言風格。

它不是傳統後端框架，不是完整前端框架，也不是資料處理函式庫。

**Syralit 負責：** UI 元件、互動狀態、事件處理、檔案上傳/下載、表格與圖表渲染、session 管理、rerun runtime、開發伺服器。

**Insyra 負責：** `DataList` / `DataTable`、CSV/JSON/Excel/SQL/Parquet 讀寫、資料清理、欄位轉換、GroupBy/Aggregate、CCL 計算、統計與資料分析。

一句話介紹：

> **Interactive data apps in Go.**
> 用 Go 快速打造互動式資料 App，並原生整合 Insyra 資料分析流程。

---

## 2. 產品目標

讓使用者用很少的 Go 程式碼完成：CSV Explorer、資料清理小工具、互動式 Dashboard、內部營運工具、分析報告 App、模型推論展示頁，以及金融／行銷／IoT 資料檢視器。

目標是讓這種程式成立（注意：新專案建議用 Insyra 的 `isr` 入口，見 §9.1）：

```go
package main

import (
	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"

	"github.com/HazelnutParadise/insyra/isr"
)

func main() {
	sy.App(func() {
		sy.Title("Sales Dashboard")

		file := sy.FileUploader("Upload CSV")
		if file == nil {
			sy.Text("Upload a CSV file to start.")
			return
		}

		// 昂貴 IO 用 Cache 包住，避免每次 rerun 重讀（見 §7）。
		dt, err := sy.Cache(file.Path, func() (*insyra.DataTable, error) {
			return isr.DT.From(isr.CSV{FilePath: file.Path}).Unwrap(), nil
		})
		if err != nil {
			sy.Error(err.Error())
			return
		}

		dt.AddColUsingCCL("profit", "['revenue'] - ['cost']")
		if e := dt.Err(); e != nil {
			sy.Warning(e.Error())
			dt.ClearErr()
		}

		syi.Table(dt)
		syi.LineChart(dt, "date", "profit")
	})
}
```

> 上例的 `isr.DT.From(...).Unwrap()` 取法以 Insyra `Docs/isr.md` 為準（本規格未逐字驗證 isr 鏈式 API 細節）。底層仍是 `insyra.ReadCSV_File`，兩者皆回傳 `*insyra.DataTable`，adapter 只認後者。`AddColUsingCCL` 回傳 `*DataTable`、錯誤走 `Err()`，已對 source 驗證（見附錄 A）。

---

## 3. 非目標

Syralit 不做：完整 React/Vue 替代品、ORM、資料分析核心、重造 DataFrame、要求使用者寫 HTML/CSS/JS、綁死 Insyra、把 AI agent 當核心必需、完整 AI agent builder、多人協作編輯、雲端部署平台。

**架構紅線：Syralit core 不得 import Insyra。** Insyra 整合一律走獨立的 adapter package（見 §4、§9）。

---

## 4. 核心架構

### 4.1 先 mono repo，後拆 module

早期就切多個獨立 Go module（core / insyra-adapter / cli 各自發 tag）會陷入版本同步地獄：改一次 core API 要同時發三個 tag。

**因此初期採單一 mono repo、多 package：**

```txt
syralit/
  cmd/syralit/              # CLI
  syralit.go ...            # core（package syralit，不 import insyra）
  integrations/insyra/      # Insyra adapter（package syinsyra）
  examples/
```

Module path：`github.com/HazelnutParadise/syralit`

core 不依賴 insyra 這條紅線，用 **CI lint 強制**（例如 `depguard`：禁止 core package import `github.com/HazelnutParadise/insyra`），而不是只靠人工自律。

等 core API 穩定後，再評估是否把 adapter / cli 拆成獨立 module。**這個決策延後，不在 MVP 處理。**

### 4.2 為什麼 adapter 要獨立

- Syralit 可以獨立存在，使用者可接其他資料來源
- 避免核心依賴膨脹
- 避免 Insyra API 變動直接破壞 Syralit core
- 同時保留第一方體驗，讓 Syralit 與 Insyra 互相加分

---

## 5. Runtime 模型

Syralit 使用 **rerun model**。`sy.App(func(){...})` 會在這些情境重新執行：頁面首次載入、widget value 改變、檔案上傳、button click、timer refresh、query param 改變、session reset。

每次 rerun：

```txt
1. 讀取目前 session state
2. 執行使用者 App function
3. 收集 UI tree
4. diff UI tree
5. 推送更新到前端
6. 保存新的 widget state
```

使用者不需要寫 route、handler、websocket、DOM 更新。

### 5.1 rerun × cache 心智模型（必讀）

這是使用者第一個會踩的坑，必須在文件首頁講清楚。

Go 是編譯型語言，但 rerun model 的語意和直覺相反：**每次事件，整個 `App` function 會從頭到尾重跑一遍**，包括裡面的讀檔、查詢、計算。Streamlit 在 Python 也是這樣，只是 Python 慢得「剛好能接受」。

因此規則是：

- **任何昂貴的東西（讀檔、SQL、模型載入、重計算）都必須用 `sy.Cache` 包住**，否則每次點按鈕都會重做一遍。
- Widget 的值、session state 會在 rerun 之間保留（見 §7），但**區域變數不會**——它們每次 rerun 重新產生。
- 想要「只在按下按鈕時才執行」的副作用，用 `if sy.Button("Run") { ... }` 包住。

正因如此，**`sy.Cache` 是 MVP 的一等公民，不是事後優化**（見 §8）。

---

## 6. 核心 API 設計

### 6.1 App 啟動

```go
sy.App(func() {
	sy.Title("My App")
})
```

進階：

```go
sy.Run(sy.Config{
	Title: "My App",
	Host:  "127.0.0.1",
	Port:  8600, // 預設不沿用 Streamlit 的 8501，避免本機並跑時撞 port
	Theme: sy.ThemeLight,
}, func() {
	sy.Title("My App")
})
```

CLI：

```bash
syralit run main.go
syralit run .
syralit new csv-explorer
```

### 6.2 文字元件

```go
sy.Title("Dashboard")
sy.Header("Section")
sy.Subheader("Upload")
sy.Text("Plain text")
sy.Textf("Count: %d", n)
sy.Markdown("**Markdown** content") // 預設 sanitize（見 §15）
sy.Code(code, "go")
sy.Caption("small text")
```

### 6.3 狀態提示

```go
sy.Success("Done")
sy.Info("Loading data")
sy.Warning("Missing column")
sy.Error("Failed to read CSV")
```

### 6.4 輸入元件

所有互動元件都接受 `sy.Key(...)`（見 §6.10 與 §13，**強烈建議一律帶 Key**）。

```go
name := sy.TextInput("Name", sy.Key("user_name"))
age := sy.NumberInput("Age", sy.NumberOptions{Min: 0, Max: 120, Value: 30})
ok := sy.Checkbox("Enable filter")
choice := sy.SelectBox("Column", []string{"A", "B", "C"})
choices := sy.MultiSelect("Columns", []string{"A", "B", "C"})
clicked := sy.Button("Run")
```

### 6.5 檔案元件

```go
file := sy.FileUploader("Upload CSV")
```

回傳：

```go
type UploadedFile struct {
	Name        string
	Path        string
	Size        int64
	ContentType string
}
```

下載：

```go
sy.DownloadFile("Download CSV", "/tmp/output.csv")
sy.DownloadBytes("Download JSON", "output.json", data)
```

### 6.6 表格元件（核心型別是介面，不是具體 struct）

**這是相對原始構想的關鍵修正。** core 的表格不吃「整張複製進來的二維陣列」，而是吃一個 **lazy 介面**，只在需要時取可視窗格。理由見 §16。

```go
// syralit core，零依賴
type TableSource interface {
	Columns() []string
	NumRows() int
	Rows(offset, limit int) [][]any // 只回傳要顯示的那一段
}

func Table(src TableSource, opts ...TableOption)
```

為了「我手上就是一小坨資料」的便利情境，core 提供一個 in-memory 實作當建構子：

```go
type Slice struct {
	Cols []string
	Data [][]any
}

func (s Slice) Columns() []string             { return s.Cols }
func (s Slice) NumRows() int                  { return len(s.Data) }
func (s Slice) Rows(off, lim int) [][]any     { /* 邊界裁切 */ }

// 用法
sy.Table(sy.Slice{
	Cols: []string{"name", "age"},
	Data: [][]any{{"Alice", 30}, {"Bob", 25}},
})
```

選項：

```go
sy.Table(src, sy.TableOptions{
	Height:      500,
	PageSize:    50,
	Searchable:  true,
	Sortable:    true,
	Selectable:  true,
	Virtualized: true,
})
```

> 設計重點：分頁／虛擬化／搜尋排序的資料供給，都透過 `TableSource.Rows(offset, limit)` 取窗格，core 永遠不需要持有整張表。Insyra adapter 提供 lazy 實作（見 §9.2），SQL cursor、檔案串流也能套同一介面。

### 6.7 圖表元件

先做常用圖：

```go
sy.LineChart(series)
sy.BarChart(series)
sy.AreaChart(series)
sy.ScatterChart(points)
```

資料模型：

```go
type SeriesModel struct {
	X    []any
	Y    []any
	Name string
}

type MultiSeriesModel struct {
	X      []any
	Series []Series
}

type Series struct {
	Name string
	Y    []any
}
```

MVP 圖表不追求完整客製化。目標是「資料 App 夠用」，不是做 BI 編輯器。

### 6.8 Metric

```go
sy.Metric("Revenue", 120000)
sy.Metric("Revenue", 120000, sy.MetricOptions{Delta: "+12%"})
```

### 6.9 Layout

```go
sy.Columns(2, func(cols []sy.Column) {
	cols[0].Write(func() { sy.Metric("Revenue", 120000) })
	cols[1].Write(func() { sy.Metric("Orders", 340) })
})

left, right := sy.Columns2()
left.Render(func()  { sy.Text("Left") })
right.Render(func() { sy.Text("Right") })

sy.Sidebar(func() {})
sy.Container(func() {})
sy.Expander("Advanced", func() {})
sy.Tabs([]string{"Data", "Chart"}, func(tab string) {})
```

### 6.10 Widget Key（一等公民）

條件式 UI（`if`、迴圈、分支）下，「按呼叫順序自動產生 ID」會在 UI 改變時錯位，導致狀態跑到錯的 widget 上。因此：

- 所有互動元件**從第一版就支援 `sy.Key(...)`**。
- **文件與範例一律示範帶 Key**，不示範省略，避免使用者 copy 範例直接中雷。
- 沒帶 Key 時才 fallback 到呼叫順序生 ID，並在 dev mode 對「條件式區塊內未帶 Key 的 widget」發出警告。

---

## 7. State 設計

```go
count := sy.State("count", 0)

if sy.Button("Add") {
	count.Set(count.Get() + 1)
}

sy.Textf("Count: %d", count.Get())
```

底層型別：

```go
type State[T any] interface {
	Get() T
	Set(value T)
	Clear()
}
```

非泛型 API：

```go
sy.Session().Set("key", value)
value, ok := sy.Session().Get("key")
```

三層：

```txt
Widget State   TextInput、SelectBox 等 UI 值
Session State  每個瀏覽器 session 私有
App Cache      可跨 session 共用（讀檔、查詢、模型載入），見 §8
```

---

## 8. Cache 設計（MVP 一等公民）

承 §5.1：rerun 下沒被 cache 的昂貴計算每次都重跑，所以 Cache 是核心而非附註。

```go
dt, err := sy.Cache("sales-data", func() (*insyra.DataTable, error) {
	return insyra.ReadCSV_File("sales.csv", false, true)
})
```

API：

```go
func Cache[T any](key string, fn func() (T, error), opts ...CacheOption) (T, error)
```

選項：

```go
sy.CacheTTL(5 * time.Minute)
sy.CacheMaxEntries(100)
sy.CacheByFileModTime("sales.csv") // 檔案更新時自動失效
```

MVP 先做 memory cache。

---

## 9. Insyra 整合規格

> 本節所有 Insyra API 簽名已對 main 分支 source 驗證，對照表見 **附錄 A**。

### 9.1 整合原則

`integrations/insyra`（package `syinsyra`，匯入別名慣例 `syi`）負責所有 Insyra 專用轉換。Syralit core 不 import insyra。

**新專案優先用 isr 入口。** Insyra 官方建議新 codebase 使用 `isr` 語法糖而非直接呼叫 main package。adapter 一律接受 `*insyra.DataTable`（isr 底層即此型別），所以兩種寫法都能餵進來。範例應同時示範 isr 入口。

### 9.2 DataTable 顯示（lazy adapter）

承 §6.6：adapter 回傳一個 `TableSource` 實作，**不把整張表複製進 core**。

```go
package syinsyra

import (
	sy "github.com/HazelnutParadise/syralit"
	"github.com/HazelnutParadise/insyra"
)

type dtSource struct{ dt *insyra.DataTable }

func (s dtSource) Columns() []string { return s.dt.Headers() }
func (s dtSource) NumRows() int      { return s.dt.NumRows() }
func (s dtSource) Rows(off, lim int) [][]any {
	all := s.dt.To2DSlice() // MVP：先全取再裁切；見下方備註
	end := off + lim
	if end > len(all) {
		end = len(all)
	}
	if off > len(all) {
		off = len(all)
	}
	return all[off:end]
}

func Table(dt *insyra.DataTable, opts ...sy.TableOption) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	sy.Table(dtSource{dt}, opts...)
}
```

> 備註：`To2DSlice()` 目前會複製整張表（已驗證：回 `[][]any`、只含資料不含表頭）。MVP 的 `dtSource` 先用它再裁切，**介面已經是 lazy 的**，所以日後把 `Rows()` 換成 Insyra 的窗格取列（若 Insyra 提供，或 adapter 自行快取一次 `To2DSlice` 結果）是**純內部優化，不破壞 core API**。這正是把介面提前到 MVP 的目的：避免日後改 API 造成破壞性變更。

### 9.3 DataTable Preview

```go
syi.Preview(dt, 20)
```

行為：顯示前 N 列、保留欄名、支援 sortable/searchable、大資料預設 virtualized。

**不要用 `dt.Show()` 的 console 輸出當 UI**——那適合 CLI，不適合瀏覽器。

### 9.4 Column Select

```go
col := syi.ColumnSelect("Choose column", dt)
// 內部：sy.SelectBox(label, dt.Headers())
```

### 9.5 圖表整合

`GetColByName(name)` 回 `*DataList`，缺欄回 `nil`（已驗證）。`DataList.Data()` 回 `[]any`（已驗證）。

```go
func LineChart(dt *insyra.DataTable, xCol, yCol string) {
	x := dt.GetColByName(xCol)
	y := dt.GetColByName(yCol)
	if x == nil || y == nil {
		sy.Warning("missing chart column")
		return
	}
	sy.LineChart(sy.SeriesModel{
		X:    x.Data(),
		Y:    y.Data(),
		Name: yCol,
	})
}
```

提供：`syi.LineChart(dt, "date", "revenue")`、`syi.BarChart(dt, "category", "sales")`、`syi.ScatterChart(dt, "x", "y")`。

> 注意：取欄一律用 `GetColByName` / `GetColByNumber`。**`GetColByIndex` 回的是欄名字串、不是 `*DataList`**（已驗證，常見誤用點）。
> MVP 不做深度型別推斷。若 y 欄不可轉數字，明確提示錯誤，不偷偷吞掉。

### 9.6 GroupBy UI

`GroupBy(keyCols...)` 回 `*GroupedDataTable`，`Aggregate(configs...)` 回 `*DataTable`（已驗證）。

進階元件：

```go
summary := syi.GroupByBuilder(dt, syi.GroupByOptions{
	DefaultKey:   "region",
	DefaultValue: "revenue",
	DefaultOp:    insyra.OpSum,
})
syi.Table(summary)
```

MVP 可先用基本 widgets 組合：

```go
key := syi.ColumnSelect("Group by", dt)
value := syi.ColumnSelect("Value", dt)
op := sy.SelectBox("Aggregate", []insyra.AggregateOp{
	insyra.OpSum,
	insyra.OpMean,
	insyra.OpCountAll,
})

summary := dt.GroupBy(key).Aggregate(insyra.AggregateConfig{
	SourceCol: value,
	Op:        op,
	As:        value + "_summary",
})
syi.Table(summary)
```

`AggregateConfig` 欄位：`SourceCol`、`As`、`Op`、`Custom`。`AggregateOp` 常數（已驗證）：`OpSum`、`OpMean`、`OpMedian`、`OpMin`、`OpMax`、`OpCount`、`OpCountAll`、`OpStdev`、`OpStdevP`、`OpVar`、`OpVarP`、`OpFirst`、`OpLast`、`OpNUnique`、`OpCustom`。

### 9.7 CCL 整合

`AddColUsingCCL(newColName, ccl)` 回 `*DataTable`，**錯誤走 `Err()` 而非回傳值**（已驗證）。CCL 欄名引用用 `['colName']`，Excel-style 用 `A`/`B`。

CCL 面板：

```go
dt = syi.CCLPanel(dt)
```

行為：輸入新欄名 → 輸入 CCL formula → 按 Apply → 執行 `dt.AddColUsingCCL(name, formula)` → 顯示 `dt.Err().Error()` → 更新表格。

```go
func CCLPanel(dt *insyra.DataTable, opts ...CCLOption) *insyra.DataTable
```

### 9.8 下載整合

`ToCSV(filePath, setRowNamesToFirstCol, setColNamesToFirstRow, includeBOM)` 回 `error`（已驗證）。

```go
func DownloadCSV(dt *insyra.DataTable, filename string) {
	tmp := sy.TempFile(filename)
	if err := dt.ToCSV(tmp, false, true, true); err != nil {
		sy.Error(err.Error())
		return
	}
	sy.DownloadFile("Download CSV", tmp)
}
```

JSON：`ToJSON_String(useColNames)` 回 `string`、`ToJSON_Bytes(useColNames)` 回 `[]byte`（已驗證）。

```go
syi.DownloadJSON(dt, "data.json")
```

---

## 10. 開發體驗

### 10.1 建立專案（約定優於配置，已實作）

```bash
syralit new my-dashboard
cd my-dashboard
go mod tidy
syralit dev          # 零 flag,熱重載
```

`syralit new` 產生**固定專案架構**(像 Streamlit 的約定),並自動 `go mod init`：

```txt
my-dashboard/
  go.mod
  main.go          # 進入點:func main() { sy.App(...) }
  syralit.toml     # 設定(port / title / theme),可選
  .gitignore
  README.md
```

進入點是 Go 慣例的 `main.go` / package main,`syralit dev` 與 `syralit run` 會自動找,**不需指定進入檔**。

**設定檔 `syralit.toml`(可選,放專案根):**

```toml
title = "Sales Dashboard"
port = 8600

[theme]
mode = "system"   # light | dark | system
accent = "#7C3AED"
radius = "12px"
```

優先序:**CLI flag > syralit.toml > 內建預設**。沒有設定檔也能跑(全用預設)。runtime(`sy.App`/`sy.Run`)與 dev supervisor 都會自動讀取工作目錄的 `syralit.toml`,所以 `go run .`、`syralit run`、`syralit dev` 三種方式設定一致。

`main.go`：

```go
package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
	sy.App(func() {
		sy.Title("My Syralit App")
		sy.Text("Hello, data.")
	})
}
```

**約定目錄:** `assets/` 存在時 `syralit dev` 自動服務並熱重載前端資產(一般 data app 不需要,前端是框架內建);`data/` 慣例放資料檔。**日常開發不需要任何 flag。**

### 10.2 Insyra 專案模板

```bash
syralit new csv-explorer --template insyra
```

`main.go` 匯入 `syralit`、`syralit/integrations/insyra`、`insyra/isr`。

### 10.3 Hot reload（已實作）

設計目標（比 air 更進一步）：改 `.go` 後瀏覽器自動更新、**對外服務全程不中斷**、編譯錯誤在前端 UI 報錯、**session state 跨重載保留**（像 Streamlit）。

```bash
syralit dev [dir]    # 熱重載 supervisor，app 永不完全停掉
syralit run [dir]    # 編譯後跑一次，不監看
```

**架構：supervisor + 子進程。** 因為 Go 是編譯型語言，改 `.go` 必須產生新 binary，不能像 Python 同進程重跑。所以由一個常駐 supervisor 持有對外 port 與瀏覽器連線（永不重啟），使用者程式跑在會被重編譯重啟的子進程：

```
browser ──WS(永不斷)──► supervisor(常駐 :8600;自服務前端 + 監看 + 編譯)
                            │  terminating WS proxy + 注入 dev 訊息
                            ▼ WS
                        app 子進程(綁內部 port,秒級重啟)
```

- **HTTP（index/JS/CSS）由 supervisor 自服務**（同一份 embed），前端永不白屏、assets 可獨立熱抽換。
- **WS 由 supervisor terminating-proxy 到子進程**：`browser↔supervisor` 連線全程不斷，只有 `supervisor↔child` 在重啟時重建。這是「不停掉 app」的關鍵——passthrough proxy 會讓瀏覽器連線跟著斷。
- **編譯先於 kill**：`go build` 成功才換子進程；失敗則**保留舊子進程繼續服務** + 推 `__dev_build_error` 給前端覆蓋層。
- **state 跨重載保留**：子進程重啟前透過 WS 控制訊息（`__dev_dump` / `__dev_state` / `__dev_restore`）把 session 的 widget 值與 State store 以 JSON dump/restore。

**取捨（必讀）：** state 跨進程序列化只能保留 **JSON-相容值**（字串、數字、bool、巢狀 map/slice）。`State[T].Get()` 內建數值型別還原（修正 JSON 把 `int` 變 `float64` 的問題）。放進 state 的複雜物件（如 `*insyra.DataTable`）**不會保留**——那類本就該放 `sy.Cache`（重載重算）。dev 模式假設單一活躍 session（本機開發一個瀏覽器分頁）；多分頁共享同一 session。

**前端 assets 熱重載：** `syralit dev --assets ./assets` 時 supervisor 從磁碟服務並監看前端資產，`.css` 變更熱抽換不刷頁，`.js` 變更自動 reload。不帶 `--assets` 時用內嵌 assets（一般使用者情境）。

dev 控制訊息協議見 `dev_common.go`。

---

## 11. 前端渲染設計

使用者不寫前端。**已採方案：極簡 vanilla client runtime。**

**採用：JSON node tree + 極簡 vanilla runtime**
server 每次 rerun 送完整 UI tree（`{type:"ui_patch", nodes:[...]}`，對齊 §13），前端一個 ~150 行的 vanilla JS runtime 負責 render + 依 widget id reconcile（保留輸入框 focus 與 caret）+ WebSocket 雙向。零 build，靠 `go:embed` 內嵌。
優點：直接對應 §13 的 stateful widget tree + diff 模型、Go 純度高、single binary、長期可控（表格/圖表重元件之後掛進同一 runtime）。
缺點：重元件（虛擬化表格、互動圖表）需自行封裝 JS bundle。

**評估過但未採用的替代：**
- **htmx + idiomorph**：上手快，但其 hypermedia / HTML-over-the-wire 哲學與「stateful widget tree + 精準 diff + focus 保留」模型有張力，§13 的 JSON 協議會用不上。
- **Preact / React**：元件生態成熟、重元件最好做，但引入 build pipeline、bundle 變大、最不像純 Go。

**重元件策略：** 表格與圖表直接封裝編譯好的 JS bundle（TanStack Table / uPlot / ECharts 擇一），用 esbuild 打包後掛進 vanilla runtime，**不暴露給使用者**。Syralit API 絕不暴露底層 JS library 細節。

```txt
shell / runtime：vanilla JS + WebSocket（go:embed，零 build）
表格：封裝的虛擬化表格元件（esbuild bundle）
圖表：封裝的 charting 元件（esbuild bundle）
UI：內建 CSS variables
```

---

## 12. Server 設計

預設 `127.0.0.1:8600`（不沿用 Streamlit 8501）。

路由：

```txt
GET  /                  App shell
GET  /assets/*          靜態資源
WS   /_syralit/ws       session websocket
POST /_syralit/upload   file upload
GET  /_syralit/download file download
GET  /api/...           使用者 opt-in 的 artifact discovery / current spec
POST /api/...           使用者 opt-in 的 agent artifact update
```

每個瀏覽器 tab 一個 session，session id 存 cookie 或 websocket init payload。

---

## 13. Event 協議

前端送出：

```json
{ "type": "widget_change", "session_id": "abc", "widget_id": "text_name", "value": "Tim" }
```

後端回傳：

```json
{ "type": "ui_patch", "nodes": [] }
```

UI node：

```go
type Node struct {
	ID       string
	Type     string
	Props    map[string]any
	Children []Node
}
```

每個 widget 需要穩定 ID。承 §6.10：未提供 Key 時依呼叫順序生 ID，但條件式 UI 下會不穩，因此所有互動元件支援 `sy.Key(...)`，且範例一律帶 Key。

---

## 14. 錯誤處理

原則：使用者程式錯誤不讓 server 無聲崩潰；單一 session panic 不影響其他 session；UI 顯示乾淨錯誤卡片；dev mode 顯示 stack trace，prod mode 隱藏。

```go
sy.Error(err.Error())
sy.Fatal(err) // 顯示錯誤、中止本次 rerun、不關閉整個 server
```

Insyra 錯誤（`Err()` 回 `*ErrorInfo`，用其 `Error()` 方法取訊息）：

```go
if e := dt.Err(); e != nil {
	sy.Warning(e.Error())
	dt.ClearErr()
}
```

> 備註：`ErrorInfo` 確定實作 `Error()`；是否另有 `Message` 等欄位本規格未驗證，故一律用 `Error()`。

---

## 15. 安全設計

預設只跑本機，不設計成公開網路服務。

限制：FileUploader 有大小限制；上傳檔存 temp dir；下載只能下載 Syralit 產生或明確允許的檔案；禁止 path traversal；Markdown 預設 sanitize；Code block 不執行；WebSocket 檢查 session。

Agent Artifact Canvas 的安全邊界：

- endpoint 預設不存在，必須由 app 明確呼叫 `HandleArtifactAPI` 或
  `HandleArtifactEndpoint` 才會註冊。
- 統一 API 只公開 app 明確傳入的 Store。`GET` 提供 discovery/current spec，
  `POST` 以 artifact ID 做 full replace；兩者都必須帶
  `Authorization: Bearer ...`。
- `POST` 必須帶最新的 `expected_revision`，revision 不符回 `409 Conflict`，
  避免多個 agent 的 full replace 靜默互相覆蓋。
- `ArtifactAPIHandler` / `ArtifactHandler` 可掛到其他 mux 或 port；這不會
  改變 Store 對 Syralit sessions 廣播更新的行為。
- Syralit 只提供 `AgentAuthenticator` / `AgentKeyStore` 介面、`StaticAgentKey`、`AgentKeyManager` UI；key 如何保存由 app 自行決定。
- Artifact DSL 只映射到受控 Syralit 元件，不接受 raw HTML、custom JS、iframe、`Component` 或內部 Node protocol。
- 所有 artifact node 必須有穩定 ID，供前端做 enter/update/exit 動畫，也避免 agent 生成不穩定樹造成狀態錯位。
- 每次成功更新會產生遞增 revision。前端必須在 keyed layout transition、
  圖表、圖片與字型完成後才把畫布標記為 `settled`；agent 必須等指定
  revision settled 後才能截圖並宣稱完成視覺驗證。
- placement 必須來自實際 render，不能為尚未出現的畫布虛構 page/selector。
  preview URL 只能來自已認證的同 origin API request 或 app 明確設定，
  不可信任任意 WebSocket `Origin`。

```go
sy.Config{
	MaxUploadSize:    100 << 20, // 100 MB
	AllowedUploadExt: []string{".csv", ".json", ".xlsx"},
}
```

CCL 是資料公式，不應執行 OS command。若未來支援 Insyra 的 Python 整合，必須獨立加安全警告。

---

## 16. 效能設計

目標規模：表格顯示最多 ~100k rows 可接受；前端只 render 可視範圍；上傳預設 100 MB；單 session 計算同步執行。

策略：

- **Table 從第一天就走 `TableSource` 介面**（§6.6），core 永不持有整張表，分頁/虛擬化天生支援。
- 大表不一次塞進 DOM；前端只取 `Rows(offset, limit)` 窗格。
- 昂貴計算用 `sy.Cache`（§8）。
- 圖表預設限制點數，超過時提示 downsample。

> 為什麼把介面提前到 MVP：`To2DSlice()` 會複製整張表，若 core 型別是 `TableModel{[][]any}`（eager），則每次 rerun 都複製 + 序列化整表，且日後要支援虛擬化必須**改 core API（破壞性變更）**。把核心型別定為 `TableSource` 介面後，eager 複製被關在 adapter 內部，未來換成真正的窗格取列只是內部優化。**這是地基，不是優化。**

---

## 17. Theme 設計

提供 light / dark / system。

```go
sy.Run(sy.Config{
	Theme: sy.Theme{Mode: "system", AccentColor: "#7C3AED", Radius: "12px"},
}, app)
```

風格：乾淨、圓角、低噪音、適合資料密集畫面、不要過度像後台管理模板。

---

## 18. 元件命名原則

Go API 要短，但不假裝是 Python。

推薦：`sy.TextInput()`、`sy.SelectBox()`、`sy.FileUploader()`、`sy.DownloadFile()`、`sy.LineChart()`、`sy.Table()`。
避免：`sy.StTextInput()`、`sy.WidgetTextInput()`、`sy.RenderDataframe()`。

Insyra adapter 用別名 `syi`：

```go
import syi "github.com/HazelnutParadise/syralit/integrations/insyra"

syi.Table(dt)
syi.Preview(dt, 20)
syi.LineChart(dt, "date", "sales")
```

---

## 19. 範例 App

- **Hello App**：Title、TextInput（帶 Key）、Button、State
- **CSV Explorer**：FileUploader、isr 讀 CSV、`syi.Table`、ColumnSelect、LineChart、DownloadCSV
- **Sales Dashboard**：Metric、GroupBy、Aggregate、BarChart、Layout
- **CCL Playground**：DataTable、AddColUsingCCL、`Err()` 顯示、重新渲染
- **SQL Viewer**：Insyra 讀 SQL 或自備資料、Table、Filter、Download

每個 example 至少能 `go test ./...` 與 `go run ./examples/<name>`。

---

## 20. 開發階段（取代版本號）

> 不綁 v0.x。以「先證明脊椎 → 再補規模 → 最後談打包」為節奏。

### MVP（脊椎）

`sy.App` / `sy.Run`、dev server、WebSocket rerun、Title/Text/Markdown、Button/TextInput/NumberInput/Checkbox/SelectBox（皆支援 Key）、FileUploader、DownloadFile、**`TableSource` 介面 + `sy.Slice` 建構子**、LineChart/BarChart、Metric、Sidebar/Columns、Session State、**`sy.Cache`（memory）**、syinsyra Table / LineChart / BarChart / DownloadCSV、CSV Explorer 與 Sales Dashboard 範例。

MVP 不做：自訂元件 SDK、AI agent builder、雲端部署、權限系統、多使用者資料庫、複雜表格編輯、雙向資料表編輯、BI drag-and-drop。

**MVP 兩個可驗證里程碑：**

```go
// 里程碑 1：rerun 脊椎
sy.App(func() {
	name := sy.TextInput("Name", sy.Key("name"))
	sy.Text("Hello, " + name)
})

// 里程碑 2：Insyra 資料流
file := sy.FileUploader("CSV")
if file != nil {
	dt, _ := insyra.ReadCSV_File(file.Path, false, true)
	syi.Table(dt)
}
```

### Next（規模與深度）

Hot reload 穩定化、Table 真窗格虛擬化（adapter 內部優化，不改 core API）、Tabs/Expander、`syi.DataFrameExplorer`、`CCLPanel`、`GroupByBuilder`、更多圖表、URL query params、更完整 cache invalidation、App config file、Agent Artifact Canvas（受控 DSL、共享畫布、opt-in endpoint、app-owned key storage hooks）。

`DataFrameExplorer` 是最重要的 Insyra 賣點：

```go
syi.DataFrameExplorer(dt)
```

功能：列數/欄數、欄位型別推測、缺失值統計、前 N 筆、摘要統計、欄位搜尋/選擇/排序、簡易篩選、下載 CSV/JSON。

### Later（打包與部署）

App packaging、single binary build、static assets embed、custom component SDK、auth hooks、background jobs、scheduled refresh、deployment docs、AI assisted dashboard generation、完整 agent builder、長期 token 稽核、多使用者 artifact 權限。

---

## 21. 最小實作順序

```txt
1.  sy.App runtime
2.  UI node tree
3.  dev server + websocket
4.  Text / Title / Button / TextInput（含 Key 機制）
5.  session state
6.  rerun
7.  TableSource 介面 + sy.Slice + Table 渲染
8.  FileUploader
9.  DownloadFile
10. LineChart / BarChart
11. syinsyra Table adapter（lazy）
12. syinsyra chart adapter
13. CSV Explorer example
14. Sidebar / Columns
15. sy.Cache
16. Sales Dashboard example
```

---

## 22. 測試規格

**Unit：** widget state、session state、rerun trigger、node tree 生成、`TableSource` 窗格邊界（offset/limit 越界裁切）、upload validation、download path safety、cache key。

**Integration：** 啟動 app、瀏覽器連線、widget change、rerun、file upload、download、websocket reconnect。

**Insyra adapter：** nil DataTable、empty DataTable、`Headers()` 對應 Columns、`To2DSlice()` 對應 Rows、`GetColByName` 缺欄回 nil、LineChart 數值欄、LineChart 非數值欄、DownloadCSV temp file、`Err()` 顯示與清除、`TableSource.Rows` 分頁正確性。

**Example：** 每個 example 能 `go test ./...` 與 `go run`。

---

## 23. 文件規格

必備：`README.md`、`SPEC.md`、`docs/getting-started.md`、`docs/widgets.md`、`docs/state.md`、`docs/rerun-and-cache.md`（§5.1 心智模型，必寫）、`docs/insyra-integration.md`、`docs/deployment.md`、`examples/README.md`。

README 第一屏：一句話定位、GIF/截圖、安裝方式、最小範例、Insyra 整合範例。

---

## 24. 發版策略

API 穩定前不承諾 semver 語意，所有變更走 changelog（`Added / Changed / Deprecated / Removed / Fixed / Security`）。等核心 API 收斂、`TableSource` 等地基型別不再變動時，再定義第一個對外發佈版本與相容性承諾。**在此之前，破壞性變更允許，但必須在 changelog 寫清楚遷移方式。**

---

## 25. 命名與品牌

專案名：**Syralit**。語感：小巧、資料導向、互動式、Go-native、不像企業後台框架。

Tagline：**Interactive data apps in Go.**（最清楚、不膨風）

adapter 命名用 `syi` / `syinsyra`，不要 `insyralit` / `syrainsyra` 這類把兩個名硬黏的命名。

---

## 26. 關鍵設計決策

1. **Syralit core 不依賴 Insyra**，且用 CI lint 強制。降耦合、避免 API 連坐、保留接其他資料來源的空間。
2. **Insyra 做官方 adapter**，保留第一方體驗，讓兩個專案互相加分。
3. **表格核心型別是 `TableSource` 介面、放 MVP 地基**，而非後期優化。避免日後為虛擬化改 core API 造成破壞性變更。
4. **`sy.Cache` 是 MVP 一等公民**，因為 rerun model 下沒 cache 的昂貴計算每次重跑。
5. **Widget Key 一等公民**，範例一律帶 Key，避免條件式 UI 狀態錯位。
6. **先 mono repo，後拆 module**，避免早期版本同步地獄。
7. **不綁版本號，用階段描述**，新專案還沒發版，硬寫 v0.x 路線圖是空中樓閣。
8. **先做單機本機 App**：安全問題少、開發體驗單純、符合此類工具初期情境。

---

## 27. 完成定義（MVP 可宣稱完成的條件）

可 `go install` CLI、`syralit new` 建專案、`syralit run` 啟動、瀏覽器可互動、widget 改變會 rerun、可上傳 CSV、可用 Insyra（含 isr 入口）讀 CSV、可顯示 Insyra DataTable（走 `TableSource`）、可畫基本圖、可下載處理後 CSV、examples 可跑、README 有完整教學、core package 通過「不得 import insyra」的 lint 檢查。

---

## 28. 總結

```txt
Syralit 管互動，Insyra 管資料，integrations/insyra 是兩者之間的官方轉接層。
core 透過 TableSource 介面消費資料，永不持有整張表，也永不 import insyra。
```

```go
sy.App(func() {
	file := sy.FileUploader("Upload CSV")
	if file == nil {
		return
	}
	dt, err := insyra.ReadCSV_File(file.Path, false, true)
	if err != nil {
		sy.Error(err.Error())
		return
	}
	syi.DataFrameExplorer(dt) // Next 階段
})
```

---

## 附錄 A：Insyra API 驗證表

對照來源：insyra `main` 分支 `Docs/DataTable.md` 與 pkg.go.dev，核對日期 **2026-06-20**。落地實作前若 Insyra 版本更新，應重新核對。

| API | 已驗證簽名 | 備註 |
|---|---|---|
| `ReadCSV_File` | `func ReadCSV_File(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool, encoding ...string) (*DataTable, error)` | ✅ |
| `(*DataTable) Headers` | `() []string` | ✅ `ColNames()` 別名 |
| `(*DataTable) To2DSlice` | `() [][]any` | ✅ **只含資料、不含表頭** |
| `(*DataTable) GetColByName` | `(name string) *DataList` | ✅ 缺欄回 nil |
| `(*DataTable) GetCol` | `(index string) *DataList` | ✅ |
| `(*DataTable) GetColByNumber` | `(index int) *DataList` | ✅ |
| `(*DataTable) GetColByIndex` | `(index string) string` | ⚠️ **回欄名字串，非 *DataList**，勿誤用 |
| `(*DataTable) Err` | `() *ErrorInfo` | ✅ |
| `(*DataTable) ClearErr` | `() *DataTable` | ✅ |
| `ErrorInfo` | 實作 `Error()` 方法 | ⚠️ 是否有 `Message` 欄位未驗證，一律用 `Error()` |
| `(*DataTable) GroupBy` | `(keyCols ...string) *GroupedDataTable` | ✅ |
| `(*GroupedDataTable) Aggregate` | `(configs ...AggregateConfig) *DataTable` | ✅ |
| `AggregateConfig` | `{ SourceCol string; As string; Op AggregateOp; Custom func(group *DataList) any }` | ✅ |
| `AggregateOp` 常數 | `OpSum OpMean OpMedian OpMin OpMax OpCount OpCountAll OpStdev OpStdevP OpVar OpVarP OpFirst OpLast OpNUnique OpCustom` | ✅ |
| `(*DataTable) ToCSV` | `(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error` | ✅ |
| `(*DataTable) ToJSON_String` | `(useColNames bool) string` | ✅ |
| `(*DataTable) ToJSON_Bytes` | `(useColNames bool) []byte` | ✅ |
| `(*DataTable) AddColUsingCCL` | `(newColName, ccl string) *DataTable` | ✅ **回 *DataTable，錯誤走 Err()**（非回 error） |
| 行列數 | `Size() (int,int)` / `NumRows() int` / `NumCols() int` / `Count() int` | ✅ `Count()` 回列數 |
| `(*DataList) Data` | `() []any` | ✅ |
| `(*DataList) Len` | `() int` | ✅ |
| `isr` 入口 | `import "github.com/HazelnutParadise/insyra/isr"`；新專案首選 | ⚠️ CSV 構造 `isr.DT.From(isr.CSV{...})` 以 `Docs/isr.md` 為準，本表未逐字驗證 |

**未驗證、落地前需確認：** `ErrorInfo` 欄位、`isr` 鏈式 API 細節、`GroupedDataTable` 的 `AggregateAll` / `Count`、CCL 完整函式清單（見 Insyra `Docs/CCL.md`）。
