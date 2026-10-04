package db

// 한국어 테스트 안내
// 예전 llm_records 테이블을 현재 기록 형태로 보완하는 마이그레이션의 멱등성을 검증한다.
// 기존 행을 유지하면서 새 컬럼/인덱스가 적용되는지 확인하므로 별도 PostgreSQL 테스트 환경에서 읽고 실행한다.

import (
	"testing"
)

// The raw-body columns were added after release, so existing installs only get
// them through llmRecordsMigrate. This runs the real migration against the dev
// PG on a table stripped back to its pre-upgrade shape, inside a transaction
// that is always rolled back — PG does transactional DDL, so nothing persists.
// 한국어 검증 목적: 이전 llm_records 구조에 원문 필드를 추가하면서 기존 행을 유지하는 마이그레이션을 확인한다.
func TestLLMRecordsMigrateAddsRawColumnsToOldTable(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()
	if err := d.EnsureLLMRecordsTable(); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	tx, err := d.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck // the test never commits

	// Roll the table back to how a pre-upgrade install looks.
	if _, err := tx.Exec(`ALTER TABLE llm_records DROP COLUMN IF EXISTS raw_request, DROP COLUMN IF EXISTS raw_response`); err != nil {
		t.Fatalf("simulate old table: %v", err)
	}
	if _, err := tx.Exec(llmRecordsMigrate); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, col := range []string{"raw_request", "raw_response"} {
		var n int
		if err := tx.QueryRow(
			`SELECT count(*) FROM information_schema.columns
			 WHERE table_name='llm_records' AND column_name=$1`, col).Scan(&n); err != nil {
			t.Fatalf("inspect %s: %v", col, err)
		}
		if n != 1 {
			t.Errorf("column %s missing after migrate", col)
		}
	}

	// Re-running must stay a no-op (the migration runs on every startup).
	if _, err := tx.Exec(llmRecordsMigrate); err != nil {
		t.Fatalf("migrate is not idempotent: %v", err)
	}

	// An insert carrying raw bodies must round-trip through the migrated table.
	var got string
	if err := tx.QueryRow(
		`INSERT INTO llm_records(model, raw_request, raw_response) VALUES ('m','{"a":1}','data: x')
		 RETURNING raw_response`).Scan(&got); err != nil {
		t.Fatalf("insert into migrated table: %v", err)
	}
	if got != "data: x" {
		t.Errorf("raw_response=%q want %q", got, "data: x")
	}
}
