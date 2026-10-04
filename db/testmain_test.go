package db

// 한국어 테스트 안내
// db 테스트 묶음의 진입점이다. agent/server 테스트와 같은 advisory lock을 사용해 공유 테스트 DB의 청소 경쟁을 줄인다.
// DB를 구성하지 않았으면 각 DB 의존 테스트가 스스로 skip한다. 테스트 성공만 보고 모든 SQL 경로를 실행했다고 판단해서는 안 된다.
// 설정된 DB에 fixture/정리 SQL이 실제 실행되므로 운영 DB 대신 독립 테스트 DB를 지정해야 한다.

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestMain acquires a PostgreSQL advisory lock (7337741002) for the entire
// db test suite. Packages agent and server hold the same lock, so parallel
// `go test ./...` runs serialize across packages on the shared dev DB and
// avoid cross-package cleanup races (e.g. DELETE FROM assets WHERE id > X
// from one package deleting assets created by another).
// 한국어 검증 목적: 테스트용 DB 구성 여부를 확인하고 패키지 간 advisory lock을 획득한 뒤 실제 테스트 묶음을 실행한다.
func TestMain(m *testing.M) {
	dsn, _, err := DSN()
	if err != nil {
		// No DB configured — tests that need PG will skip themselves.
		os.Exit(m.Run())
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil || conn.Ping() != nil {
		os.Exit(m.Run())
	}
	defer conn.Close()
	if _, err := conn.Exec(`SELECT pg_advisory_lock(7337741002)`); err != nil {
		os.Exit(m.Run())
	}
	defer conn.Exec(`SELECT pg_advisory_unlock(7337741002)`) //nolint:errcheck
	os.Exit(m.Run())
}
