package backtest

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

const maxCandlesPerRequest = 200

// DataLoader fetches historical candles from an exchange adapter and caches
// them in a SQLite database. If the adapter is nil it operates in cache-only
// mode.
type DataLoader struct {
	db      *sql.DB
	adapter adapter.TradingAdapter
}

// NewDataLoader creates a DataLoader. adp may be nil for cache-only mode.
func NewDataLoader(db *sql.DB, adp adapter.TradingAdapter) *DataLoader {
	return &DataLoader{db: db, adapter: adp}
}

// LoadCandles returns candles for the given symbol/timeframe in [from, to].
//  1. Try the SQLite cache first.
//  2. If the cache is empty or insufficient and an adapter is available, fetch
//     from the exchange with automatic pagination (max 200 candles per request).
//  3. Fetched candles are written back to the cache.
func (d *DataLoader) LoadCandles(ctx context.Context, symbol, timeframe string, from, to time.Time) ([]adapter.Candle, error) {
	// 1. Query cache
	cached, err := d.loadFromCache(ctx, symbol, timeframe, from, to)
	if err != nil {
		return nil, fmt.Errorf("cache read: %w", err)
	}

	// Determine expected candle count for the range.
	dur := timeframeToDuration(timeframe)
	if dur == 0 {
		return nil, fmt.Errorf("unsupported timeframe %q", timeframe)
	}
	expected := int(to.Sub(from)/dur) + 1

	if len(cached) >= expected || d.adapter == nil {
		return cached, nil
	}

	// 2. Fetch from exchange with pagination
	fetched, err := d.fetchFromExchange(ctx, symbol, timeframe, from, to, dur)
	if err != nil {
		return nil, fmt.Errorf("exchange fetch: %w", err)
	}

	// 3. Cache fetched candles
	if err := d.writeToCache(ctx, symbol, timeframe, fetched); err != nil {
		return nil, fmt.Errorf("cache write: %w", err)
	}

	// Filter to requested range
	var result []adapter.Candle
	for _, c := range fetched {
		if !c.Timestamp.Before(from) && !c.Timestamp.After(to) {
			result = append(result, c)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

func (d *DataLoader) loadFromCache(_ context.Context, symbol, timeframe string, from, to time.Time) ([]adapter.Candle, error) {
	rows, err := d.db.Query(
		`SELECT timestamp, open, high, low, close, volume
		 FROM candle_cache
		 WHERE symbol = ? AND timeframe = ? AND timestamp >= ? AND timestamp <= ?
		 ORDER BY timestamp ASC`,
		symbol, timeframe, from.Unix(), to.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candles []adapter.Candle
	for rows.Next() {
		var ts int64
		var c adapter.Candle
		if err := rows.Scan(&ts, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			return nil, err
		}
		c.Timestamp = time.Unix(ts, 0).UTC()
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

func (d *DataLoader) fetchFromExchange(ctx context.Context, symbol, timeframe string, from, to time.Time, dur time.Duration) ([]adapter.Candle, error) {
	totalNeeded := int(to.Sub(from)/dur) + 1
	var all []adapter.Candle

	for len(all) < totalNeeded {
		limit := totalNeeded - len(all)
		if limit > maxCandlesPerRequest {
			limit = maxCandlesPerRequest
		}

		batch, err := d.adapter.GetCandles(ctx, symbol, timeframe, limit)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		all = append(all, batch...)

		if len(batch) < limit {
			break
		}
	}

	return all, nil
}

func (d *DataLoader) writeToCache(_ context.Context, symbol, timeframe string, candles []adapter.Candle) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO candle_cache (symbol, timeframe, timestamp, open, high, low, close, volume)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range candles {
		if _, err := stmt.Exec(symbol, timeframe, c.Timestamp.Unix(), c.Open, c.High, c.Low, c.Close, c.Volume); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// timeframeToDuration converts timeframe strings like "1m", "5m", "1h", "4h",
// "1d", "1w" to their corresponding time.Duration.
func timeframeToDuration(tf string) time.Duration {
	if len(tf) < 2 {
		return 0
	}

	unit := tf[len(tf)-1]
	numStr := strings.TrimRight(tf[:len(tf)-1], " ")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}

	switch unit {
	case 'm':
		return time.Duration(num) * time.Minute
	case 'h':
		return time.Duration(num) * time.Hour
	case 'd':
		return time.Duration(num) * 24 * time.Hour
	case 'w':
		return time.Duration(num) * 168 * time.Hour
	default:
		return 0
	}
}
