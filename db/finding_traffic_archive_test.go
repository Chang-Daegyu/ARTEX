package db

// 한국어 테스트 안내
// 증거용 advisory lock 키 충돌 방지와 구형 아카이브에서 없는 발견 버전/증거 연결의 기본값 복원을 검증한다.
// 예전 형식에 없는 증거 버전도 일관된 기본값으로 복구되어 보고서 신선도 판정이 깨지지 않아야 한다.

import (
	"encoding/json"
	"fmt"
	"testing"
)

// 한국어 검증 목적: 증거 파일/참조를 조율하는 advisory lock이 스키마·테스트·회사 잠금과 다른 키인지 확인한다.
func TestFindingEvidenceLockNamespace(t *testing.T) {
	for _, reserved := range []int64{7337741001, 7337741002, 7337741003} {
		if findingEvidenceLockKey == reserved {
			t.Fatal("evidence lock collides with migration/test/company lock")
		}
	}
}

// 한국어 검증 목적: v1/v2 아카이브에 없는 증거 테이블/버전 필드를 복원 시 빈 연결과 0 버전으로 보완하는지 확인한다.
func TestFindingTrafficLegacyArchiveDefaults(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err = d.EnsureLLMRecordsTable(); err != nil {
		t.Fatal(err)
	}
	if err = d.EnsureLLMUsageTable(); err != nil {
		t.Fatal(err)
	}
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			task, err := d.CreateTask("legacy archive evidence defaults", "fixture", nil, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { d.Exec(`DELETE FROM task_archives WHERE task_id=$1`, task.ID); d.DeleteTask(task.ID) }()
			f, err := d.Exploration(task.ExplorationID).RecordFinding(t.Context(), RecordFindingInput{TaskID: task.ID, ExplorationID: task.ExplorationID, Summary: "legacy", Severity: "low"})
			if err != nil {
				t.Fatal(err)
			}
			if err = d.SetPaused(task.ID, true); err != nil {
				t.Fatal(err)
			}
			archive, err := d.QueueTaskArchive(task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = d.ClaimTaskArchiveJob(t.Context()); err != nil {
				t.Fatal(err)
			}
			snapshot, err := d.SnapshotTaskArchive(task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err = d.CompleteTaskArchive(archive.ID, snapshot, "/tmp/legacy-test.tar.zst", "fixture", 1, 1); err != nil {
				t.Fatal(err)
			}
			snapshot.FormatVersion = version
			delete(snapshot.Tables, "finding_traffic_bindings")
			delete(snapshot.Tables, "traffic_evidence_snapshots")
			var rows []map[string]any
			if err = json.Unmarshal(snapshot.Tables["findings"], &rows); err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				delete(row, "evidence_version")
				delete(row, "report_evidence_version")
			}
			snapshot.Tables["findings"], _ = json.Marshal(rows)
			if _, err = d.QueueTaskArchiveRestore(archive.ID); err != nil {
				t.Fatal(err)
			}
			if _, err = d.ClaimTaskArchiveJob(t.Context()); err != nil {
				t.Fatal(err)
			}
			if _, err = d.RestoreTaskArchive(archive.ID, snapshot, 0); err != nil {
				t.Fatal(err)
			}
			got, err := d.GetFindingTraffic(t.Context(), f.FindingID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Version != 0 || got.ReportVersion != 0 || len(got.Bindings) != 0 {
				t.Fatal(got)
			}
		})
	}
}
