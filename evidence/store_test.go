// [한국어 파일 안내] evidence/store_test.go
// PostgreSQL·임시 traffic·독립 evidence 디렉터리를 함께 사용하는 증거 통합 테스트다.
// ARTEX_PG_DSN이 없으면 건너뛰며, 지정할 때는 테스트 전용 DB가 필요하다. 실제 행 생성·수정·삭제가 발생한다.
// 증거 바인딩/버전·원시 삭제 후 보존·손상/실패 원자성·중복 동시 요청·GC 유예·복원 잠금을 검증한다.
package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/traffic"
)

// 한국어 해설: 명시적으로 구성한 PostgreSQL에 테스트 작업을 만들고 임시 수집/증거 디렉터리와 종료 정리를 준비한다.
func evidenceFixture(t *testing.T) (*Store, db.RecordFindingInput, string) {
	t.Helper()
	dsn := os.Getenv("ARTEX_PG_DSN")
	if dsn == "" {
		t.Skip("ARTEX_PG_DSN is required for evidence integration tests")
	}
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pg.Close() })
	task, err := pg.CreateTask("evidence integration "+t.Name(), "local fixtures", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pg.DeleteFindingsByTask(task.ID); pg.DeleteTask(task.ID) })
	dir := t.TempDir()
	tr, err := traffic.Open(filepath.Join(dir, "traffic"), ":0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.Close() })
	store := New(pg, tr, filepath.Join(dir, "evidence"))
	in := db.RecordFindingInput{TaskID: task.ID, ExplorationID: task.ExplorationID, Worker: "test", VulnClass: "TEST", Name: "Evidence fixture", Severity: "low", Summary: "local test"}
	return store, in, dir
}

// 한국어 해설: 인라인 또는 외부 blob 형태의 합성 교환을 넣어 증거 Reader가 미리보기 대신 전체 바이트를 복사하는지 시험한다.
func seedExchange(t *testing.T, s *Store, id string, body []byte, spill bool) {
	t.Helper()
	var blob any
	inline := body
	if spill {
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		blob = hash
		path := filepath.Join(filepath.Dir(s.Dir), "traffic", "_blobs", "sha256", hash[:2], hash+".bin")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		inline = []byte("TRUNCATED PREVIEW")
	}
	_, err := s.Traffic.DB().Exec(`INSERT INTO exchanges(id,ts,host,method,url_template,url,status,content_type,req_len,resp_len,path) VALUES(?,?,'fixture.local','POST','/test',?,200,'application/octet-stream',2,?,'')`, id, time.Now().Unix(), "https://fixture.local/"+id, len(body))
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Traffic.DB().Exec(`INSERT INTO exchange_bodies(id,req_head,req_body,resp_head,resp_body,resp_blob) VALUES(?,?,?, ?,?,?)`, id, "POST /test HTTP/1.1\nHost: fixture.local\n", []byte("{}"), "HTTP 200\nContent-Type: application/octet-stream\n", inline, blob)
	if err != nil {
		t.Fatal(err)
	}
}

// 한국어 해설: 발견 기록·중복 바인딩·공유 스냅샷·원시 삭제 뒤 읽기·보고서 버전 충돌·공유 본문 GC 보존을 한 흐름으로 검증한다.
func TestEvidenceBindingLifecycle(t *testing.T) {
	s, in, _ := evidenceFixture(t)
	ctx := context.Background()
	large := bytes.Repeat([]byte{0, 1, 2, 255, 'a', 'b'}, 180000)
	for i := 0; i < 3; i++ {
		seedExchange(t, s, fmt.Sprint(i), large, i == 2)
	}
	refs := []db.TrafficRef{{TrafficID: "0", Role: "baseline", Note: "normal"}, {TrafficID: "1", Role: "proof", Note: "proof"}, {TrafficID: "2", Role: "verification", Note: "binary"}}
	r, err := s.Record(ctx, in, refs)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Traffic.Bindings) != 3 || r.Traffic.Version != 1 {
		t.Fatalf("record=%+v", r)
	}
	again, err := s.Bind(ctx, r.FindingID, []db.TrafficRef{{TrafficID: "1", Note: "must not overwrite"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Bindings) != 3 || again.Version != 1 || again.Bindings[1].Note != "proof" {
		t.Fatalf("duplicate changed bindings: %+v", again)
	}
	other, err := s.Record(ctx, in, []db.TrafficRef{refs[2]})
	if err != nil {
		t.Fatal(err)
	}
	if other.Traffic.Bindings[0].SnapshotID != r.Traffic.Bindings[2].SnapshotID {
		t.Fatal("snapshot was not shared")
	}
	if _, err = s.Traffic.DeleteHost("fixture.local"); err != nil {
		t.Fatal(err)
	}
	for _, binding := range r.Traffic.Bindings {
		if err = s.WithBinding(ctx, r.FindingID, binding.ID, func(b db.FindingTrafficBinding) error {
			f, _, err := s.OpenBody(b.Snapshot, "response")
			if err != nil {
				return err
			}
			defer f.Close()
			got, err := io.ReadAll(f)
			if !bytes.Equal(got, large) {
				t.Fatal("body was truncated/changed")
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	v := again.Version
	if _, err = s.DB.SetFindingReportVersionByNodeID(ctx, r.NodeID, "report", &v); err != nil {
		t.Fatal(err)
	}
	note := "updated"
	if err = s.DB.EditFindingTraffic(ctx, r.FindingID, again.Bindings[0].ID, v, nil, &note, false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.SetFindingReportVersionByNodeID(ctx, r.NodeID, "stale report", &v); !errors.Is(err, db.ErrEvidenceConflict) {
		t.Fatalf("stale report: %v", err)
	}
	f, err := s.DB.GetFinding(r.FindingID)
	if err != nil {
		t.Fatal(err)
	}
	if f.Report != "report" || f.EvidenceVersion == f.ReportEvidenceVersion || f.TrafficCount != 3 {
		t.Fatalf("finding versions: %+v", f)
	}
	if err = s.DB.EditFindingTraffic(ctx, r.FindingID, again.Bindings[0].ID, v, nil, nil, true, nil); !errors.Is(err, db.ErrEvidenceConflict) {
		t.Fatalf("stale delete: %v", err)
	}
	if _, err = s.DB.SetFindingReportByNodeID(r.NodeID, "legacy report"); err != nil {
		t.Fatal(err)
	}
	f, err = s.DB.GetFinding(r.FindingID)
	if err != nil {
		t.Fatal(err)
	}
	if f.ReportEvidenceVersion != -1 {
		t.Fatal("legacy write claimed current evidence")
	}
	if _, err = s.DB.DeleteFinding(r.FindingID); err != nil {
		t.Fatal(err)
	}
	if err = s.Collect(ctx, time.Now().Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = s.WithBinding(ctx, other.FindingID, other.Traffic.Bindings[0].ID, func(b db.FindingTrafficBinding) error {
		f, _, err := s.OpenBody(b.Snapshot, "response")
		if f != nil {
			f.Close()
		}
		return err
	}); err != nil {
		t.Fatal("GC removed shared body:", err)
	}
}

// 한국어 해설: 없는 ID·잘린 본문·디스크 실패·DB 실패에 finding/노드가 부분 기록되지 않고 과거/no-traffic 경로도 유지되는지 확인한다.
func TestEvidenceAtomicFailuresAndLegacy(t *testing.T) {
	s, in, dir := evidenceFixture(t)
	ctx := context.Background()
	seedExchange(t, s, "valid", []byte("OK"), false)
	counts := func() (findings, nodes int) {
		s.DB.QueryRow(`SELECT count(*) FROM findings WHERE task_id=$1`, in.TaskID).Scan(&findings)
		s.DB.QueryRow(`SELECT count(*) FROM exploration_nodes WHERE exploration_id=$1 AND kind='finding'`, in.ExplorationID).Scan(&nodes)
		return
	}
	for _, kind := range []string{"missing_id", "missing_body", "disk_failure", "database_failure"} {
		t.Run(kind, func(t *testing.T) {
			input := in
			refs := []db.TrafficRef{{TrafficID: "valid"}, {TrafficID: "absent"}}
			local := *s
			switch kind {
			case "missing_body":
				seedExchange(t, s, "broken", []byte("more than preview"), false)
				s.Traffic.DB().Exec(`UPDATE exchange_bodies SET resp_body=? WHERE id='broken'`, []byte("x"))
				refs = []db.TrafficRef{{TrafficID: "valid"}, {TrafficID: "broken"}}
			case "disk_failure":
				local.Dir = filepath.Join(dir, "file-not-dir")
				if err := os.WriteFile(local.Dir, []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
				refs = refs[:1]
			case "database_failure":
				input.AssetIDs = []int64{9223372036854775807}
				refs = refs[:1]
			}
			if _, err := local.Record(ctx, input, refs); err == nil {
				t.Fatal("expected failure")
			}
			if f, n := counts(); f != 0 || n != 0 {
				t.Fatalf("partial record: findings=%d nodes=%d", f, n)
			}
		})
	}
	legacy := filepath.Join(dir, "traffic", "legacy")
	os.MkdirAll(legacy, 0o700)
	os.WriteFile(filepath.Join(legacy, "request.http"), []byte("GET / HTTP/1.1\nHost: old.local\n"), 0o600)
	os.WriteFile(filepath.Join(legacy, "response.http"), []byte("HTTP 200\n\nlegacy body"), 0o600)
	_, err := s.Traffic.DB().Exec(`INSERT INTO exchanges(id,ts,host,method,url_template,url,status,content_type,req_len,resp_len,path) VALUES('old',1,'old.local','GET','/','http://old.local/',200,'text/plain',0,11,'legacy')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Record(ctx, in, []db.TrafficRef{{TrafficID: "old"}}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Record(ctx, in, nil); err != nil {
		t.Fatal("legacy no-traffic report failed:", err)
	}
}

// 한국어 해설: 동일 참조 동시 바인딩이 하나로 합쳐지고 원시 트래픽 삭제와 경쟁해도 이미 연결된 원문이 읽히는지 검증한다.
func TestEvidenceConcurrentBindAndDelete(t *testing.T) {
	s, in, _ := evidenceFixture(t)
	ctx := context.Background()
	seedExchange(t, s, "race", bytes.Repeat([]byte("large"), 90000), true)
	r, err := s.Record(ctx, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Bind(ctx, r.FindingID, []db.TrafficRef{{TrafficID: "race"}})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.DB.GetFindingTraffic(ctx, r.FindingID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Bindings) != 1 || list.Version != 1 {
		t.Fatalf("concurrent duplicates: %+v", list)
	}
	// Race a new capture copy with source deletion: either the full copy commits
	// or nothing does. An already-bound snapshot remains readable in both cases.
	wg.Add(2)
	go func() { defer wg.Done(); s.Bind(ctx, r.FindingID, []db.TrafficRef{{TrafficID: "race"}}) }()
	go func() { defer wg.Done(); s.Traffic.DeleteHost("fixture.local") }()
	wg.Wait()
	if err = s.WithBinding(ctx, r.FindingID, list.Bindings[0].ID, func(b db.FindingTrafficBinding) error {
		f, _, err := s.OpenBody(b.Snapshot, "response")
		if f != nil {
			f.Close()
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// 한국어 해설: 23시간에는 유지하고 25시간 뒤 무참조 파일을 수거하며, 진행 중 복원은 잠금으로 보호하고 바뀐 메타데이터 해시는 거절하는지 확인한다.
func TestEvidenceGCGraceAndActiveRestore(t *testing.T) {
	s, in, _ := evidenceFixture(t)
	ctx := context.Background()
	seedExchange(t, s, "gc", []byte("unique unreferenced gc body"), false)
	f, err := s.Record(ctx, in, []db.TrafficRef{{TrafficID: "gc"}})
	if err != nil {
		t.Fatal(err)
	}
	snap := f.Traffic.Bindings[0].Snapshot
	archive := t.TempDir()
	if err = s.CopySnapshots(ctx, []db.TrafficEvidenceSnapshot{snap}, archive); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.DeleteFinding(f.FindingID); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err = s.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}
	path, err := hashPath(s.Dir, snap.RespHash)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Collect(ctx, now.Add(23*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("GC grace ignored", err)
	}
	if err = s.Collect(ctx, now.Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("orphan was not collected", err)
	}
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		finished <- s.WithInstalledSnapshots(ctx, []db.TrafficEvidenceSnapshot{snap}, archive, func() error { close(entered); <-release; return nil })
	}()
	<-entered
	short, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	err = s.Collect(short, now.Add(72*time.Hour))
	cancel()
	close(release)
	if err == nil {
		t.Fatal("GC entered during active restore")
	}
	if err = <-finished; err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("active restore lost body", err)
	}
	// A portable package cannot substitute metadata while retaining its old ID.
	bad := snap
	bad.URL = "http://tampered.local/"
	if err = s.InstallSnapshots(ctx, []db.TrafficEvidenceSnapshot{bad}, archive); err == nil {
		t.Fatal("accepted tampered snapshot")
	}
}
