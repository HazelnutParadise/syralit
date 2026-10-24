// insyra-candlestick draws an interactive candlestick chart with technical
// indicators (Insyra quant/ta) from a table of open/high/low/close/volume.
// Upload your own CSV (Big5 broker exports work) or use the synthetic sample.
package main

import (
	"math"
	"math/rand"
	"time"

	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"

	"github.com/HazelnutParadise/insyra"
)

func sample() *insyra.DataTable {
	rng := rand.New(rand.NewSource(7))
	var date, open, high, low, closeP, vol []any
	price := 100.0
	day := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	for len(date) < 160 {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			day = day.AddDate(0, 0, 1)
			continue
		}
		o := price
		c := o * (1 + (rng.Float64()-0.48)*0.03)
		date = append(date, day.Format("2006-01-02"))
		open = append(open, math.Round(o*100)/100)
		closeP = append(closeP, math.Round(c*100)/100)
		high = append(high, math.Round(math.Max(o, c)*(1+rng.Float64()*0.01)*100)/100)
		low = append(low, math.Round(math.Min(o, c)*(1-rng.Float64()*0.01)*100)/100)
		vol = append(vol, 1000+rng.Intn(5000))
		price = c
		day = day.AddDate(0, 0, 1)
	}
	return insyra.NewDataTable(
		insyra.NewDataList(date...).SetName("date"), insyra.NewDataList(open...).SetName("open"),
		insyra.NewDataList(high...).SetName("high"), insyra.NewDataList(low...).SetName("low"),
		insyra.NewDataList(closeP...).SetName("close"), insyra.NewDataList(vol...).SetName("volume"))
}

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Candlestick"), sy.ConfigIcon("🕯️"))
		sy.Title("K 線與技術指標")

		dt := syi.UploadTable("上傳 OHLCV CSV（欄位 date/open/high/low/close/volume）")
		if dt == nil {
			sy.Caption("尚未上傳，顯示模擬資料。")
			dt = sample()
		}

		ind := syi.Indicators{}
		if sy.Checkbox("均線 SMA 5 / 20", sy.DefaultValue(true)) {
			ind.SMA = []int{5, 20}
		}
		if sy.Checkbox("布林通道 (20, 2)") {
			ind.Bollinger = &syi.BollingerSpec{N: 20, K: 2}
		}
		if sy.Checkbox("MACD (12, 26, 9)", sy.DefaultValue(true)) {
			ind.MACD = &syi.MACDSpec{Fast: 12, Slow: 26, Signal: 9}
		}
		if sy.Checkbox("KD (9)", sy.DefaultValue(true)) {
			ind.KD = 9
		}
		if sy.Checkbox("RSI (14)") {
			ind.RSI = 14
		}
		if sy.Checkbox("乖離率 (10)") {
			ind.Bias = []int{10}
		}

		syi.CandlestickChart(dt, syi.OHLCV{
			Date: "date", Open: "open", High: "high", Low: "low", Close: "close", Volume: "volume",
		}, ind)
	})
}
