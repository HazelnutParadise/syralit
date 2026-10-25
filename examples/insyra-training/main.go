// insyra-training trains a small Insyra neural network (insyra/nn) in the
// background and shows a live progress bar and loss curve with syinn.FitLive.
package main

import (
	"math"

	sy "github.com/HazelnutParadise/syralit"
	syinn "github.com/HazelnutParadise/syralit/integrations/insyra/nnview"

	"github.com/HazelnutParadise/insyra/nn"
)

// problem builds y = sin(3a) + b over a grid of (a, b) points.
func problem() (*nn.Tensor, *nn.Tensor) {
	const n = 4096
	xd, yd := make([]float32, 0, n*2), make([]float32, 0, n)
	for i := 0; i < n; i++ {
		a, b := float64(i%64)/64, float64(i/64)/64
		xd = append(xd, float32(a), float32(b))
		yd = append(yd, float32(math.Sin(3*a)+b))
	}
	x, _ := nn.NewTensor([]int{n, 2}, xd)
	y, _ := nn.NewTensor([]int{n, 1}, yd)
	return x, y
}

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Training"), sy.ConfigIcon("🧠"))
		sy.Title("神經網路訓練（Insyra nn）")

		epochs := int(sy.Slider("Epochs", 5, 200, sy.DefaultValue(60), sy.Step(5), sy.Key("epochs")))
		if !sy.Button("開始訓練", sy.Key("start")) && !sy.State("started", false).Get() {
			sy.Caption("按下按鈕開始，訓練在背景進行，畫面不會卡住。")
			return
		}
		sy.State("started", false).Set(true)

		x, y := problem()
		model, err := nn.NewSequential(nn.NewTape(1), nn.Dense(2, 64), nn.ReLU(), nn.Dense(64, 64), nn.ReLU(), nn.Dense(64, 1))
		if err != nil {
			sy.Exception(err)
			return
		}
		h := syinn.FitLive("demo", model, x, y, nn.FitConfig{
			Epochs: epochs, BatchSize: 32, Seed: 1, Loss: nn.MSE{}, Optimizer: nn.Adam{Rate: 0.01}, Quiet: true,
		})
		if h.Done() && h.Err() == nil {
			sy.Success("訓練完成")
		}
	})
}
