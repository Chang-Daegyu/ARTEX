// [한국어 파일 안내] evidence/testmain_test.go
// 증거 통합 테스트 묶음의 PostgreSQL 초기화와 공통 suite advisory lock을 관리한다.
// ARTEX_PG_DSN이 없는 일반 검사는 각 테스트의 skip으로 끝날 수 있으므로 통합 검증 성공과 구분한다.
// 설정된 DB에는 초기화와 쓰기가 발생하므로 테스트 전용 DB만 지정한다.
package evidence

import (
	"context"
	"os"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// Initialize an explicitly configured fresh database before taking the same
// suite lock as db, agent and server. Hold it on one pinned connection.
// 한국어 해설: DSN이 없는 경로는 일반 테스트 진입, DSN이 있으면 DB를 여는 suite 실행 경로로 분기한다.
func TestMain(m *testing.M) {
	if os.Getenv("ARTEX_PG_DSN") == "" {
		os.Exit(m.Run())
	}
	os.Exit(runEvidenceSuite(m))
}
// 한국어 해설: 한 고정 연결에서 다른 DB 사용 패키지와 같은 advisory lock을 유지한 채 m.Run을 실행한다.
func runEvidenceSuite(m *testing.M) int {
	pg, err := db.Open(os.Getenv("ARTEX_PG_DSN"))
	if err != nil {
		panic(err)
	}
	defer pg.Close()
	conn, err := pg.Conn(context.Background())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(context.Background(), `SELECT pg_advisory_lock(7337741002)`); err != nil {
		panic(err)
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(7337741002)`)
	return m.Run()
}
