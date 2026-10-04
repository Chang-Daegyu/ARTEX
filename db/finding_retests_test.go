package db

// 한국어 테스트 안내
// 동시 재검증 요청 중복 억제, 원본 스냅샷/대화 생성, 결과 판정과 완료 상태의 조합을 검증한다.
// 취소/실패한 실행의 fixed 잠정 결과가 원래 발견을 수정 완료로 바꾸지 않는지, 재시작 복구가 미완료 상태를 정리하는지도 확인한다.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

func retestDB(t *testing.T) (*DB, int64) {
	t.Helper()
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	fid, err := d.AddFinding(0, 0, "retest-test", "测试漏洞", "high", "original summary", "original evidence", "worker", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = d.Exec(`DELETE FROM conversations WHERE id IN (SELECT conversation_id FROM finding_retests WHERE finding_id=$1)`, fid)
		_, _ = d.DeleteFinding(fid)
		d.Close()
	})
	return d, fid
}

// 한국어 검증 목적: 동시 재검증 요청이 하나의 활성 대화/실행으로 합쳐지고 원본 이력이 유지되는지 확인한다.
func TestRetestAtomicDeduplicationAndHistory(t *testing.T) {
	d, fid := retestDB(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	createdCount := 0
	ids := make([]int64, 0, 8)
	for range 8 {
		wg.Go(func() {
			r, c, created, err := d.CreateFindingRetest(t.Context(), fid, "  修复版本 v2  ")
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			ids = append(ids, r.ID)
			if created {
				createdCount++
				if c == nil {
					t.Error("created without conversation")
				}
			}
		})
	}
	wg.Wait()
	if createdCount != 1 || len(ids) != 8 {
		t.Fatalf("created=%d ids=%v", createdCount, ids)
	}
	for _, id := range ids {
		if id != ids[0] {
			t.Fatal("duplicate active retests", ids)
		}
	}
	rows, err := d.ListFindingRetests(fid)
	if err != nil || len(rows) != 1 {
		t.Fatalf("history=%v err=%v", rows, err)
	}
	r := rows[0]
	if r.Snapshot != nil {
		t.Fatal("history leaks large snapshot")
	}
	ctx := t.Context()
	full, err := d.FindingRetestForConversation(ctx, *r.ConversationID)
	if err != nil || !strings.Contains(string(full.Snapshot), "original evidence") {
		t.Fatalf("snapshot=%+v err=%v", full, err)
	}
	var messages int
	if err := d.QueryRow(`SELECT count(*) FROM conversation_activities WHERE conversation_id=$1 AND kind='user'`, *r.ConversationID).Scan(&messages); err != nil || messages != 1 {
		t.Fatalf("messages=%d err=%v", messages, err)
	}
	_, _ = d.Exec(`UPDATE findings SET evidence='changed evidence' WHERE id=$1`, fid)
	full, err = d.FindingRetestForConversation(ctx, *r.ConversationID)
	if err != nil || strings.Contains(string(full.Snapshot), "changed evidence") {
		t.Fatal("snapshot changed", err)
	}
	if err = d.RecordFindingRetestResult(ctx, *r.ConversationID, "fixed", "summary", "proof"); !errors.Is(err, ErrRetestNotRunning) {
		t.Fatal("pending accepted result", err)
	}
	if ok, err := d.StartFindingRetest(ctx, r.ID); err != nil || !ok {
		t.Fatalf("start=%t %v", ok, err)
	}
	if err := d.RecordFindingRetestResult(ctx, 0, "fixed", "summary", "proof"); !errors.Is(err, ErrRetestNotRunning) {
		t.Fatal("unscoped write accepted", err)
	}
	for range 2 {
		if err := d.RecordFindingRetestResult(ctx, *r.ConversationID, "fixed", "修复验证通过", "正常对照可用，原触发条件失效"); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.RecordFindingRetestResult(ctx, *r.ConversationID, "reproduced", "different", "proof"); !errors.Is(err, ErrRetestNotRunning) {
		t.Fatal("overwrote staged result", err)
	}
	if err := d.FinishFindingRetest(r.ID, "completed", ""); err != nil {
		t.Fatal(err)
	}
	if err := d.RecordFindingRetestResult(ctx, *r.ConversationID, "fixed", "summary", "proof"); !errors.Is(err, ErrRetestNotRunning) {
		t.Fatal("overwrote sealed result", err)
	}
	f, _ := d.GetFinding(fid)
	if f.Status != FindingFixed || f.Evidence != "changed evidence" || f.Report != "" {
		t.Fatal("fixed retest did not update only triage", f)
	}
	next, _, created, err := d.CreateFindingRetest(ctx, fid, "second")
	if err != nil || !created || next.ID == r.ID {
		t.Fatalf("new history=%+v created=%t err=%v", next, created, err)
	}
	if err := d.FinishFindingRetest(next.ID, "completed", ""); err != nil {
		t.Fatal(err)
	}
	rows, _ = d.ListFindingRetests(fid)
	if len(rows) != 2 || rows[0].Status != "failed" || rows[0].Error == "" || rows[1].Verdict != "fixed" {
		t.Fatal("missing verdict treated as successful", rows)
	}
}

// 한국어 검증 목적: 대화 삭제나 재시작 뒤 실행기 없는 미완료 retest가 계속 활성으로 남지 않는지 확인한다.
func TestRetestDeletionAndRestart(t *testing.T) {
	d, fid := retestDB(t)
	r, c, _, err := d.CreateFindingRetest(t.Context(), fid, "delete")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.DeleteConversation(c.ID); err != nil {
		t.Fatal(err)
	}
	rows, _ := d.ListFindingRetests(fid)
	if len(rows) != 1 || rows[0].ConversationID != nil || rows[0].Status != "stopped" {
		t.Fatal(rows)
	}
	next, c2, created, err := d.CreateFindingRetest(t.Context(), fid, "restart")
	if err != nil || !created {
		t.Fatal("deleted session blocks retry", err)
	}
	if ok, err := d.StartFindingRetest(t.Context(), next.ID); err != nil || !ok {
		t.Fatal(err)
	}
	if err := d.RecoverFindingRetests(); err != nil {
		t.Fatal(err)
	}
	if err := d.FinishFindingRetest(next.ID, "completed", ""); err != nil {
		t.Fatal(err)
	}
	full, _ := d.FindingRetestForConversation(t.Context(), c2.ID)
	if full.Status != "stopped" || full.FinishedAt == nil {
		t.Fatal("restart result overwritten", full)
	}
	if _, err := d.DeleteFinding(fid); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = d.QueryRow(`SELECT count(*) FROM finding_retests WHERE id IN ($1,$2)`, r.ID, next.ID).Scan(&count)
	if count != 0 {
		t.Fatal("finding deletion did not cascade")
	}
	_ = d.DeleteConversation(c2.ID)
}

// 한국어 검증 목적: 잘못된 재검증 입력/저장 실패가 부분 대화나 반쪽 retest를 남기지 않는지 확인한다.
func TestRetestValidationAndRollback(t *testing.T) {
	d, fid := retestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, _, err := d.CreateFindingRetest(ctx, fid, ""); err == nil {
		t.Fatal("cancelled creation succeeded")
	}
	for _, args := range [][3]string{{"unknown", "summary", "proof"}, {"fixed", " ", "proof"}, {"fixed", "summary", ""}, {"fixed", strings.Repeat("x", 16001), "proof"}} {
		if err := d.RecordFindingRetestResult(t.Context(), 0, args[0], args[1], args[2]); err == nil {
			t.Fatal("bad result accepted", args[0])
		}
	}
	r, c, _, err := d.CreateFindingRetest(t.Context(), fid, "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	full, err := d.FindingRetestForConversation(t.Context(), c.ID)
	var snap map[string]json.RawMessage
	if err != nil || json.Unmarshal(full.Snapshot, &snap) != nil || len(snap["finding"]) == 0 {
		t.Fatal("invalid snapshot", err)
	}
	if r.Notes != "snapshot" {
		t.Fatal("notes lost")
	}
}

// 한국어 검증 목적: fixed라는 잠정 결과만으로 처분 상태를 바꾸지 않고 실제 성공 완료가 필요한지 확인한다.
func TestRetestFixedTriageOnlyAfterSuccessfulCompletion(t *testing.T) {
	for _, tc := range []struct{ verdict, terminal, want string }{
		{"fixed", "completed", FindingFixed},
		{"fixed", "failed", FindingInProgress},
		{"fixed", "stopped", FindingInProgress},
		{"reproduced", "completed", FindingInProgress},
		{"inconclusive", "completed", FindingInProgress},
		{"", "completed", FindingInProgress},
	} {
		t.Run(tc.verdict+"/"+tc.terminal, func(t *testing.T) {
			d, fid := retestDB(t)
			if _, err := d.SetFindingStatus(fid, FindingInProgress); err != nil {
				t.Fatal(err)
			}
			r, c, _, err := d.CreateFindingRetest(t.Context(), fid, "check triage")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.StartFindingRetest(t.Context(), r.ID); err != nil {
				t.Fatal(err)
			}
			if tc.verdict != "" {
				if err := d.RecordFindingRetestResult(t.Context(), c.ID, tc.verdict, "summary", "proof"); err != nil {
					t.Fatal(err)
				}
			}
			f, err := d.GetFinding(fid)
			if err != nil || f.Status != FindingInProgress {
				t.Fatal("staged verdict changed triage", err, f)
			}
			if err := d.FinishFindingRetest(r.ID, tc.terminal, ""); err != nil {
				t.Fatal(err)
			}
			f, err = d.GetFinding(fid)
			if err != nil || f.Status != tc.want || f.Evidence != "original evidence" {
				t.Fatalf("finding=%+v err=%v want=%s", f, err, tc.want)
			}
			// Re-delivering completion must not undo a later user decision.
			if _, err := d.SetFindingStatus(fid, FindingIgnored); err != nil {
				t.Fatal(err)
			}
			if err := d.FinishFindingRetest(r.ID, "completed", ""); err != nil {
				t.Fatal(err)
			}
			f, err = d.GetFinding(fid)
			if err != nil || f.Status != FindingIgnored {
				t.Fatal("replayed completion overwrote triage", err, f)
			}
		})
	}
}
