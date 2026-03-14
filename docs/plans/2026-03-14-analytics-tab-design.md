# Analytics Tab Design

## Goal
Add a new "Analytics" tab to the web dashboard with portfolio analytics widgets inspired by Perplexity's financial dashboard, but tailored for multi-asset trading (Crypto, Forex, Stocks, DeFi).

## Architecture
New tab in Sidebar navigation, rendering an `AnalyticsView` component in App.tsx. Same inline-style patterns as existing dashboard. All data is mock/procedural for now (like existing components).

## Layout
2-column CSS Grid, 3 rows:

```
Row 1: [Portfolio vs Benchmark chart  ] [Risk Analysis panel]
Row 2: [Asset Allocation donut        ] [Monthly Returns heatmap] [Top Movers]
Row 3: [Drawdown Chart - full width                              ]
```

- gridTemplateColumns: `1fr 380px` (row 1), `1fr 1fr 1fr` (row 2), `1fr` (row 3)
- Same gap (12px), card styling, and CSS variables as main dashboard

## Components

### 1. PerformanceChart
- SVG line chart comparing portfolio equity curve vs benchmark
- Benchmark selector: S&P 500, BTC, Custom
- Time range: 1M, 3M, 6M, 1Y, ALL
- Two lines with area fill, legend

### 2. RiskAnalysis
- 5 metrics in a vertical card:
  - Sharpe Ratio (with color: >1 green, <1 yellow, <0 red)
  - Sortino Ratio
  - Annualized Volatility %
  - Max Drawdown %
  - Top Concentration % (largest position weight)

### 3. AssetAllocation
- SVG donut chart with segments per asset class
- Legend: Crypto, Forex, Stocks, DeFi with % and dollar values
- Center text: total portfolio value

### 4. MonthlyReturns
- Grid heatmap: columns = months (Jan-Dec), rows = years
- Cell color: green gradient for positive, red gradient for negative
- Hover tooltip with exact % return

### 5. TopMovers
- Two sections: Best Performers, Worst Performers (top 3 each)
- Symbol, return %, small sparkline

### 6. DrawdownChart
- SVG area chart showing drawdown % over time
- Shaded red area below zero line
- Annotations for max drawdown point

## Sidebar Update
Add "Analytics" tab between "Trades" and "Strategy" with chart-bar icon.
