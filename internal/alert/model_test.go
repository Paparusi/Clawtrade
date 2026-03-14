package alert

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func testDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		symbol TEXT,
		condition TEXT NOT NULL,
		threshold REAL,
		expression TEXT,
		message TEXT,
		enabled BOOLEAN DEFAULT 1,
		one_shot BOOLEAN DEFAULT 0,
		last_triggered_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alert_id INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		value REAL,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAlertStore_CreateAndList(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{
		Type:      AlertTypePrice,
		Symbol:    "BTC/USDT",
		Condition: CondAbove,
		Threshold: 70000,
		Message:   "BTC above 70k",
		Enabled:   true,
		OneShot:   false,
	}

	id, err := store.Create(a)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	alerts, err := store.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Symbol != "BTC/USDT" {
		t.Errorf("expected BTC/USDT, got %s", alerts[0].Symbol)
	}
}

func TestAlertStore_Delete(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "ETH/USDT", Condition: CondBelow, Threshold: 3000, Enabled: true}
	id, _ := store.Create(a)

	if err := store.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	alerts, _ := store.ListEnabled()
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts after delete, got %d", len(alerts))
	}
}

func TestAlertStore_Disable(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "ETH/USDT", Condition: CondAbove, Threshold: 5000, Enabled: true, OneShot: true}
	id, _ := store.Create(a)

	if err := store.Disable(id); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	alerts, _ := store.ListEnabled()
	if len(alerts) != 0 {
		t.Fatalf("expected 0 enabled alerts, got %d", len(alerts))
	}
}

func TestAlertStore_LogTrigger(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true}
	id, _ := store.Create(a)

	if err := store.LogTrigger(id, "price.update", 71000, "BTC crossed 70k"); err != nil {
		t.Fatalf("LogTrigger: %v", err)
	}

	history, err := store.TodayHistory()
	if err != nil {
		t.Fatalf("TodayHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
}

func TestAlertStore_UpdateLastTriggered(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true}
	id, _ := store.Create(a)

	now := time.Now()
	if err := store.UpdateLastTriggered(id, now); err != nil {
		t.Fatalf("UpdateLastTriggered: %v", err)
	}
}
