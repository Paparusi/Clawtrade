package security

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/clawtrade/clawtrade/internal/database"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestAuditLog_WriteAndVerify(t *testing.T) {
	db := setupTestDB(t)
	audit := NewAuditLog(db)

	err := audit.Log("agent:trader", "PLACE_ORDER", map[string]any{
		"symbol": "BTC/USDT", "side": "BUY", "size": 0.1,
	}, "RSI oversold")
	if err != nil {
		t.Fatal(err)
	}

	err = audit.Log("agent:trader", "PLACE_ORDER", map[string]any{
		"symbol": "ETH/USDT", "side": "SELL", "size": 1.0,
	}, "Taking profit")
	if err != nil {
		t.Fatal(err)
	}

	valid, err := audit.VerifyChain()
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Error("chain should be valid")
	}
}

func TestAuditLog_TamperDetection(t *testing.T) {
	db := setupTestDB(t)
	audit := NewAuditLog(db)

	audit.Log("agent", "ACTION1", nil, "")
	audit.Log("agent", "ACTION2", nil, "")

	db.Exec("UPDATE audit_log SET action='TAMPERED' WHERE id=1")

	valid, err := audit.VerifyChain()
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Error("tampered chain should be invalid")
	}
}
