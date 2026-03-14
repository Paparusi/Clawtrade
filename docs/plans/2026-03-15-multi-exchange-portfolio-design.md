# Multi-Exchange Portfolio Aggregation Design

**Goal:** Aggregate balances and positions from all connected exchanges into a unified portfolio view with per-exchange breakdown, accessible via API, WebSocket events, agent tools, and frontend dashboard.

**Architecture:** Extend existing `streaming/portfolio_poller.go` to poll ALL connected adapters (not just the first), merge results with per-exchange tagging. Add `GET /api/v1/portfolio` endpoint. Update agent tools for multi-exchange awareness. Update frontend PortfolioSummary with exchange breakdown.

**Tech Stack:** Go (backend), existing adapter infrastructure, EventBus, React/TypeScript (frontend).

---

## Backend Changes

### 1. Extend PortfolioPoller

Current: polls first connected adapter only.
New: iterate ALL adapters in parallel, collect balances/positions from each, tag with exchange name.

Enhanced `portfolio.update` event data:
```json
{
  "balances": [...],
  "positions": [...],
  "total_pnl": 1234.56,
  "exchanges": {
    "binance": {"total": 5000.0, "balances": [...], "positions": [...]},
    "bybit":   {"total": 3000.0, "balances": [...], "positions": [...]}
  }
}
```

### 2. New API Endpoint

`GET /api/v1/portfolio` — returns aggregated portfolio snapshot (same structure as event). Queries all connected adapters on-demand.

Existing `/api/v1/balances?exchange=X` and `/api/v1/positions?exchange=X` remain unchanged.

### 3. Update Agent Tools

- `get_balances` without `exchange` → aggregate from all exchanges
- `get_balances` with `exchange` → single exchange (current behavior)
- Same for `get_positions`

## Frontend Changes

### 4. Update PortfolioSummary

Add per-exchange breakdown below the 4 stat cards. Small colored pills: "Binance: $5,000 · Bybit: $3,000". Data from `exchanges` field in `portfolio.update` event.

### 5. Update API Client

Add `fetchPortfolio()` calling `GET /api/v1/portfolio`.
