package db

// 한국어 테스트 안내
// 스키마 실행이 PostgreSQL deadlock(40P01)만 재시도하는지와 Open의 내장 데이터 등록/재호출 멱등성을 검증한다.
// testDSN은 설정이 없으면 DB 의존 테스트를 건너뛴다. 통과 표시에 skip이 섞였는지 확인해야 실제 DB 검증 범위를 알 수 있다.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type schemaExecFunc func(context.Context, string, ...any) (sql.Result, error)

func (f schemaExecFunc) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return f(ctx, query, args...)
}

// 한국어 검증 목적: 재시도 가능한 40P01과 즉시 반환해야 하는 일반 오류를 구별하고 지연 순서를 검사한다.
func TestApplySchemaRetriesOnlyDeadlocks(t *testing.T) {
	deadlockAttempts := 0
	delays := []time.Duration{}
	err := applySchemaWithRetry(t.Context(), schemaExecFunc(func(context.Context, string, ...any) (sql.Result, error) {
		deadlockAttempts++
		if deadlockAttempts < 3 {
			return nil, &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
		}
		return nil, nil
	}), func(delay time.Duration) {
		delays = append(delays, delay)
	})
	if err != nil {
		t.Fatal(err)
	}
	if deadlockAttempts != 3 {
		t.Fatalf("schema attempts=%d, want 3", deadlockAttempts)
	}
	if len(delays) != 2 || delays[0] != schemaDeadlockRetryDelays[0] || delays[1] != schemaDeadlockRetryDelays[1] {
		t.Fatalf("schema retry delays=%v", delays)
	}

	nonRetryable := errors.New("permission denied")
	nonRetryableAttempts := 0
	err = applySchemaWithRetry(t.Context(), schemaExecFunc(func(context.Context, string, ...any) (sql.Result, error) {
		nonRetryableAttempts++
		return nil, nonRetryable
	}), func(time.Duration) {
		t.Fatal("non-deadlock error must not be retried")
	})
	if !errors.Is(err, nonRetryable) || nonRetryableAttempts != 1 {
		t.Fatalf("non-retryable result: attempts=%d err=%v", nonRetryableAttempts, err)
	}
}

// testDSN returns the configured DSN, skipping the test when neither the env var
// nor a config file supplies one (DSN no longer has a built-in default).
func testDSN(t *testing.T) string {
	t.Helper()
	dsn, _, err := DSN()
	if err != nil {
		t.Skipf("no database config (%v) — skipping", err)
	}
	return dsn
}

// TestOpenSeed opens the live dev PG, applies schema, seeds, and verifies the
// builtin agents + their variable catalog exist. Skips if PG is unreachable.
// 한국어 검증 목적: 실제 DB 초기화 후 내장 에이전트/변수가 존재하고 다시 열어도 중복 행이 생기지 않는지 확인한다.
func TestOpenSeed(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	wantAgents := len(builtinAgents)
	var agents int
	if err := d.QueryRow(`SELECT count(*) FROM agents WHERE builtin`).Scan(&agents); err != nil {
		t.Fatal(err)
	}
	if agents != wantAgents {
		t.Fatalf("want %d builtin agents, got %d", wantAgents, agents)
	}

	// planner should have at least its seeded catalog vars
	wantPlannerVars := 0
	for _, a := range builtinAgents {
		if a.key == "planner" {
			wantPlannerVars = len(a.vars)
			break
		}
	}
	var plannerVars int
	if err := d.QueryRow(`
SELECT count(*) FROM agent_prompt_vars v
JOIN agents a ON a.id = v.agent_id
WHERE a.key = 'planner'`).Scan(&plannerVars); err != nil {
		t.Fatal(err)
	}
	if plannerVars < wantPlannerVars {
		t.Fatalf("want at least %d planner vars, got %d", wantPlannerVars, plannerVars)
	}

	// re-open must be idempotent (no duplicate agents)
	d2, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	if err := d2.QueryRow(`SELECT count(*) FROM agents WHERE builtin`).Scan(&agents); err != nil {
		t.Fatal(err)
	}
	if agents != wantAgents {
		t.Fatalf("after re-open want %d agents, got %d", wantAgents, agents)
	}
}
