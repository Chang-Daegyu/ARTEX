// [한국어 길잡이] manager_lifecycle_test.go의 테스트 읽기
// 상태 스냅샷이 내부 슬라이스를 복사하고 여러 goroutine에서 서로 다른 시점의 필드가 섞이지 않게 읽히는지 검사한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestTaskLifecycleSnapshotCopiesContextSlices, TestTaskLifecycleSnapshotConcurrentConsistency.
// 이 테스트의 fixture와 단언을 함께 읽으면 일반 경로 외의 예외·경쟁 조건을 이해할 수 있다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"fmt"
	"sync"
	"testing"
)

func TestTaskLifecycleSnapshotCopiesContextSlices(t *testing.T) {
	task := &Task{
		Status:        "running",
		SourceTaskIDs: []int64{11, 12},
		CompanyIDs:    []int64{21, 22},
	}

	snapshot := task.lifecycleSnapshot()
	snapshot.SourceTaskIDs[0] = 99
	snapshot.CompanyIDs[0] = 98

	fresh := task.lifecycleSnapshot()
	if got := fresh.SourceTaskIDs[0]; got != 11 {
		t.Fatalf("snapshot source IDs alias task storage: got %d", got)
	}
	if got := fresh.CompanyIDs[0]; got != 21 {
		t.Fatalf("snapshot company IDs alias task storage: got %d", got)
	}

	sources := []int64{31, 32}
	companies := []int64{41, 42}
	task.updateLifecycle(func(state *taskLifecycleState) {
		state.SourceTaskIDs = sources
		state.CompanyIDs = companies
	})
	sources[0] = 97
	companies[0] = 96

	fresh = task.lifecycleSnapshot()
	if got := fresh.SourceTaskIDs[0]; got != 31 {
		t.Fatalf("lifecycle update retained caller source slice: got %d", got)
	}
	if got := fresh.CompanyIDs[0]; got != 41 {
		t.Fatalf("lifecycle update retained caller company slice: got %d", got)
	}
}

func TestTaskLifecycleSnapshotConcurrentConsistency(t *testing.T) {
	task := &Task{}
	writeState := func(generation int64, queued bool) {
		task.updateLifecycle(func(state *taskLifecycleState) {
			state.Status = map[bool]string{true: "queued", false: "paused"}[queued]
			state.Queued = queued
			state.Paused = !queued
			state.QueuedAt = generation
			state.QueueMode = map[bool]string{true: "resume", false: "bootstrap"}[queued]
			state.CompletedAt = generation
			state.FirstRunAt = generation
			state.DeadlineAt = generation
			state.SourceTaskIDs = []int64{generation, generation + 1}
			state.CompanyIDs = []int64{generation, generation + 1}
		})
	}
	writeState(1, true)

	var wg sync.WaitGroup
	for writer := 0; writer < 4; writer++ {
		writer := writer
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 1; i <= 1000; i++ {
				generation := int64(writer*1000 + i)
				writeState(generation, generation%2 == 0)
			}
		}()
	}

	errCh := make(chan error, 4)
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				state := task.lifecycleSnapshot()
				if state.Queued == state.Paused {
					errCh <- fmt.Errorf("queued=%v paused=%v", state.Queued, state.Paused)
					return
				}
				wantStatus := map[bool]string{true: "queued", false: "paused"}[state.Queued]
				wantMode := map[bool]string{true: "resume", false: "bootstrap"}[state.Queued]
				if state.Status != wantStatus || state.QueueMode != wantMode {
					errCh <- fmt.Errorf("inconsistent status/mode: %+v", state)
					return
				}
				generation := state.QueuedAt
				if state.CompletedAt != generation || state.FirstRunAt != generation || state.DeadlineAt != generation ||
					len(state.SourceTaskIDs) != 2 || len(state.CompanyIDs) != 2 ||
					state.SourceTaskIDs[0] != generation || state.CompanyIDs[0] != generation {
					errCh <- fmt.Errorf("mixed lifecycle snapshot: %+v", state)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
