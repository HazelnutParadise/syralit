package main

import (
	"fmt"
	"strings"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

// ── Sample Data ─────────────────────────────────────────────────────────────

type Product struct {
	Name     string
	Category string
	Year     int
	Month    int
	Price    float64
	Quantity int
	Revenue  float64
}

func buildSampleData() []Product {
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Home"}
	names := map[string][]string{
		"Electronics": {"Laptop", "Phone", "Tablet", "Headphones", "Camera"},
		"Clothing":    {"T-Shirt", "Jacket", "Jeans", "Shoes", "Hat"},
		"Food":        {"Coffee", "Tea", "Snack Box", "Energy Bar", "Juice"},
		"Books":       {"Go in Action", "Clean Code", "Design Patterns", "Refactoring", "The Pragmatic Programmer"},
		"Home":        {"Desk Lamp", "Plant Pot", "Pillow", "Mug Set", "Candle"},
	}

	// Base prices by item
	prices := map[string]float64{
		"Laptop": 1200, "Phone": 800, "Tablet": 500, "Headphones": 150, "Camera": 900,
		"T-Shirt": 25, "Jacket": 120, "Jeans": 65, "Shoes": 90, "Hat": 20,
		"Coffee": 12, "Tea": 8, "Snack Box": 15, "Energy Bar": 3, "Juice": 5,
		"Go in Action": 45, "Clean Code": 40, "Design Patterns": 55, "Refactoring": 50, "The Pragmatic Programmer": 48,
		"Desk Lamp": 35, "Plant Pot": 18, "Pillow": 30, "Mug Set": 22, "Candle": 14,
	}

	// Quantity multipliers per month (simulate seasonality)
	monthMultipliers := []float64{0.8, 0.75, 0.9, 1.0, 1.1, 1.2, 1.0, 0.95, 1.05, 1.15, 1.4, 1.6}

	var data []Product
	for _, year := range []int{2024, 2025} {
		for month := 1; month <= 12; month++ {
			for _, cat := range categories {
				for _, name := range names[cat] {
					price := prices[name]
					baseQty := int(price/10) + 5
					qty := int(float64(baseQty) * monthMultipliers[month-1])
					if year == 2025 {
						qty = int(float64(qty) * 1.15) // 15% YoY growth
					}
					data = append(data, Product{
						Name:     name,
						Category: cat,
						Year:     year,
						Month:    month,
						Price:    price,
						Quantity: qty,
						Revenue:  price * float64(qty),
					})
				}
			}
		}
	}
	return data
}

// ── Helpers ─────────────────────────────────────────────────────────────────

var monthLabels = []string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func filterData(data []Product, year int, categories []string) []Product {
	var out []Product
	for _, p := range data {
		if p.Year == year && contains(categories, p.Category) {
			out = append(out, p)
		}
	}
	return out
}

func formatCurrency(v float64) string {
	if v >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", v/1_000_000)
	}
	if v >= 1_000 {
		return fmt.Sprintf("$%.1fK", v/1_000)
	}
	return fmt.Sprintf("$%.0f", v)
}

// ── Pages ───────────────────────────────────────────────────────────────────

func init() {
	sy.AddPage("Overview", overviewPage, sy.PageIcon("📊"), sy.PageOrder(1))
	sy.AddPage("Data", dataPage, sy.PageIcon("🗃️"), sy.PageOrder(2))
	sy.AddPage("Analysis", analysisPage, sy.PageIcon("🔬"), sy.PageOrder(3))
}

func main() {
	sy.App(nil)
}

// ── Page 1: Overview ────────────────────────────────────────────────────────

func overviewPage() {
	sy.SetPageConfig(sy.PageTitle("Data Explorer"), sy.ConfigIcon("📊"))

	allData := sy.CacheData("all_products", buildSampleData, sy.TTL(10*time.Minute))

	sy.Title("Sales Overview")
	sy.Caption("Product sales dashboard with hardcoded sample data")

	// Compute KPIs for 2025
	var totalRevenue, totalQty float64
	var productCount int
	categorySet := map[string]bool{}
	for _, p := range allData {
		if p.Year == 2025 {
			totalRevenue += p.Revenue
			totalQty += float64(p.Quantity)
			categorySet[p.Category] = true
			productCount++
		}
	}

	// Compare to 2024
	var prevRevenue float64
	for _, p := range allData {
		if p.Year == 2024 {
			prevRevenue += p.Revenue
		}
	}
	revDelta := (totalRevenue - prevRevenue) / prevRevenue * 100

	cols := sy.Columns(4)
	cols[0](func() {
		sy.Metric("Total Revenue (2025)", formatCurrency(totalRevenue), sy.Delta(fmt.Sprintf("+%.1f%%", revDelta)))
	})
	cols[1](func() {
		sy.Metric("Units Sold", fmt.Sprintf("%.0f", totalQty), sy.Delta("+15%"))
	})
	cols[2](func() {
		sy.Metric("Avg Order Value", formatCurrency(totalRevenue/totalQty))
	})
	cols[3](func() {
		sy.Metric("Categories", fmt.Sprintf("%d", len(categorySet)))
	})

	sy.Divider()

	// Monthly revenue line chart (2024 vs 2025)
	sy.Header("Monthly Revenue Trend")
	monthly2024 := make([]float64, 12)
	monthly2025 := make([]float64, 12)
	for _, p := range allData {
		if p.Year == 2024 {
			monthly2024[p.Month-1] += p.Revenue
		} else {
			monthly2025[p.Month-1] += p.Revenue
		}
	}
	sy.LineChart(map[string][]float64{
		"2024": monthly2024,
		"2025": monthly2025,
	}, sy.ChartTitle("Revenue by Month"))

	// Category distribution pie chart
	lr := sy.WeightedColumns(3, 2)
	lr[0](func() {
		sy.Subheader("Revenue by Category (2025)")
		catRevenue := map[string]float64{}
		for _, p := range allData {
			if p.Year == 2025 {
				catRevenue[p.Category] += p.Revenue
			}
		}
		sy.PieChart(catRevenue, sy.ChartTitle("Category Share"))
	})
	lr[1](func() {
		sy.Subheader("Top 5 Products by Revenue")
		type prodSum struct {
			Name    string
			Revenue float64
		}
		prodMap := map[string]float64{}
		for _, p := range allData {
			if p.Year == 2025 {
				prodMap[p.Name] += p.Revenue
			}
		}
		// Find top 5
		var sorted []prodSum
		for name, rev := range prodMap {
			sorted = append(sorted, prodSum{name, rev})
		}
		// Simple selection sort for top 5
		for i := 0; i < len(sorted) && i < 5; i++ {
			maxIdx := i
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].Revenue > sorted[maxIdx].Revenue {
					maxIdx = j
				}
			}
			sorted[i], sorted[maxIdx] = sorted[maxIdx], sorted[i]
		}
		if len(sorted) > 5 {
			sorted = sorted[:5]
		}
		rows := make([][]any, len(sorted))
		for i, ps := range sorted {
			rows[i] = []any{ps.Name, formatCurrency(ps.Revenue)}
		}
		sy.DataFrame([]string{"Product", "Revenue"}, rows)
	})
}

// ── Page 2: Data ────────────────────────────────────────────────────────────

func dataPage() {
	allData := sy.CacheData("all_products", buildSampleData, sy.TTL(10*time.Minute))

	sy.Title("Data Explorer")

	// Sidebar filters
	var selectedYear string
	var selectedCategories []string

	sy.Sidebar(func() {
		sy.Header("Filters")

		selectedYear = sy.SelectBox("Year", []string{"2025", "2024"}, sy.Key("filter_year"))
		selectedCategories = sy.MultiSelect(
			"Categories",
			[]string{"Electronics", "Clothing", "Food", "Books", "Home"},
			sy.Key("filter_categories"),
		)
	})

	// Parse year
	year := 2025
	if selectedYear == "2024" {
		year = 2024
	}

	// Default to all categories if none selected
	if len(selectedCategories) == 0 {
		selectedCategories = []string{"Electronics", "Clothing", "Food", "Books", "Home"}
	}

	filtered := filterData(allData, year, selectedCategories)

	sy.Subheader(fmt.Sprintf("Showing %d records for %d", len(filtered), year))
	sy.Caption("Categories: " + strings.Join(selectedCategories, ", "))

	sy.Divider()

	// DataFrame view
	sy.Header("Sales Data")
	rows := make([][]any, len(filtered))
	for i, p := range filtered {
		rows[i] = []any{p.Name, p.Category, monthLabels[p.Month-1], p.Price, p.Quantity, formatCurrency(p.Revenue)}
	}
	sy.DataFrame(
		[]string{"Product", "Category", "Month", "Price", "Qty", "Revenue"},
		rows,
		sy.Height(400),
	)

	sy.Divider()

	// Editable summary table
	sy.Header("Category Summary (Editable)")
	catSummary := map[string][2]float64{} // [revenue, qty]
	for _, p := range filtered {
		s := catSummary[p.Category]
		s[0] += p.Revenue
		s[1] += float64(p.Quantity)
		catSummary[p.Category] = s
	}
	editHeaders := []string{"Category", "Revenue", "Units", "Target Met"}
	var editRows [][]any
	for _, cat := range selectedCategories {
		if s, ok := catSummary[cat]; ok {
			met := s[0] > 100000
			editRows = append(editRows, []any{cat, s[0], s[1], met})
		}
	}
	edited := sy.DataEditor(
		editHeaders,
		editRows,
		sy.Key("category_editor"),
		sy.ColConfig(map[string]sy.ColumnConfig{
			"Revenue":    {Type: "number", Min: 0},
			"Units":      {Type: "number", Min: 0},
			"Target Met": {Type: "checkbox"},
		}),
	)
	if len(edited) > 0 {
		sy.Caption(fmt.Sprintf("Edited %d rows", len(edited)))
	}

	sy.Divider()

	// Export button
	var csvBuilder strings.Builder
	csvBuilder.WriteString("Product,Category,Month,Price,Quantity,Revenue\n")
	for _, p := range filtered {
		csvBuilder.WriteString(fmt.Sprintf("%s,%s,%s,%.2f,%d,%.2f\n",
			p.Name, p.Category, monthLabels[p.Month-1], p.Price, p.Quantity, p.Revenue))
	}
	sy.DownloadButton(
		"Export CSV",
		[]byte(csvBuilder.String()),
		fmt.Sprintf("sales_%d.csv", year),
		sy.MimeType("text/csv"),
	)
}

// ── Page 3: Analysis ────────────────────────────────────────────────────────

func analysisPage() {
	allData := sy.CacheData("all_products", buildSampleData, sy.TTL(10*time.Minute))

	sy.Title("Sales Analysis")
	sy.Caption("Cross-category comparisons and correlations")

	// Use 2025 data for analysis
	data2025 := filterData(allData, 2025, []string{"Electronics", "Clothing", "Food", "Books", "Home"})

	// Radar chart: category performance across dimensions
	sy.Header("Category Performance Radar")
	sy.Caption("Comparing categories across Revenue, Volume, Avg Price, Product Count, and Growth")

	catMetrics := map[string]struct {
		revenue  float64
		qty      float64
		prices   float64
		count    int
		maxPrice float64
	}{}
	for _, p := range data2025 {
		m := catMetrics[p.Category]
		m.revenue += p.Revenue
		m.qty += float64(p.Quantity)
		m.prices += p.Price
		m.count++
		if p.Price > m.maxPrice {
			m.maxPrice = p.Price
		}
		catMetrics[p.Category] = m
	}

	// Normalize to 0-100 scale for radar
	radarLabels := []string{"Revenue", "Volume", "Avg Price", "Product Range", "Max Ticket"}
	radarData := map[string][]float64{}

	// Find max values for normalization
	var maxRev, maxQty, maxAvg, maxCount, maxTicket float64
	for _, m := range catMetrics {
		avgP := m.prices / float64(m.count)
		if m.revenue > maxRev {
			maxRev = m.revenue
		}
		if m.qty > maxQty {
			maxQty = m.qty
		}
		if avgP > maxAvg {
			maxAvg = avgP
		}
		if float64(m.count) > maxCount {
			maxCount = float64(m.count)
		}
		if m.maxPrice > maxTicket {
			maxTicket = m.maxPrice
		}
	}

	for cat, m := range catMetrics {
		avgP := m.prices / float64(m.count)
		radarData[cat] = []float64{
			m.revenue / maxRev * 100,
			m.qty / maxQty * 100,
			avgP / maxAvg * 100,
			float64(m.count) / maxCount * 100,
			m.maxPrice / maxTicket * 100,
		}
	}
	sy.RadarChart(radarLabels, radarData, sy.ChartTitle("Category Comparison"))

	sy.Divider()

	// Bar chart: top 10 products by revenue
	sy.Header("Top 10 Products by Revenue")
	prodRevenue := map[string]float64{}
	for _, p := range data2025 {
		prodRevenue[p.Name] += p.Revenue
	}

	type kv struct {
		Name string
		Rev  float64
	}
	var prodList []kv
	for name, rev := range prodRevenue {
		prodList = append(prodList, kv{name, rev})
	}
	// Sort descending
	for i := 0; i < len(prodList); i++ {
		maxIdx := i
		for j := i + 1; j < len(prodList); j++ {
			if prodList[j].Rev > prodList[maxIdx].Rev {
				maxIdx = j
			}
		}
		prodList[i], prodList[maxIdx] = prodList[maxIdx], prodList[i]
	}
	if len(prodList) > 10 {
		prodList = prodList[:10]
	}

	barValues := make([]float64, len(prodList))
	barLabels := make([]string, len(prodList))
	for i, item := range prodList {
		barValues[i] = item.Rev
		barLabels[i] = item.Name
	}
	sy.BarChart(map[string][]float64{
		"Revenue": barValues,
	}, sy.ChartTitle("Top Products"))

	sy.Divider()

	// Scatter chart: price vs total quantity sold
	sy.Header("Price vs Quantity Sold")
	sy.Caption("Each point represents a product; size correlates with revenue")

	prodQty := map[string]float64{}
	prodPrice := map[string]float64{}
	for _, p := range data2025 {
		prodQty[p.Name] += float64(p.Quantity)
		prodPrice[p.Name] = p.Price
	}

	scatterByCat := map[string][][2]float64{}
	seen := map[string]bool{}
	for _, p := range data2025 {
		if seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		scatterByCat[p.Category] = append(scatterByCat[p.Category], [2]float64{
			prodPrice[p.Name],
			prodQty[p.Name],
		})
	}
	sy.ScatterChart(scatterByCat, sy.ChartTitle("Price vs Quantity by Category"))

	sy.Divider()

	// Monthly breakdown bar chart by category
	sy.Header("Monthly Revenue by Category")
	monthlyCat := map[string][]float64{}
	for _, cat := range []string{"Electronics", "Clothing", "Food", "Books", "Home"} {
		monthlyCat[cat] = make([]float64, 12)
	}
	for _, p := range data2025 {
		monthlyCat[p.Category][p.Month-1] += p.Revenue
	}
	sy.BarChart(monthlyCat, sy.ChartTitle("Monthly Revenue Breakdown"))
}
