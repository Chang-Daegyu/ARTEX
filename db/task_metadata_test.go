package db

// 한국어 테스트 안내
// 작업 고정/해제·이름 수정·목록 순서와 대화 일괄 삭제의 실제 삭제 ID 반환을 검증한다.
// 메타데이터 수정이 실행 상태와 섞이지 않는지, 없는 ID가 성공 목록에 잘못 포함되지 않는지 확인한다.

import (
	"fmt"
	"testing"
	"time"
)

// 한국어 검증 목적: 작업 고정/이름 변경과 목록의 안정적인 순서가 의도대로 적용되는지 확인한다.
func TestTaskPinOrderingAndRename(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	suffix := time.Now().UnixNano()
	first, err := d.CreateTask(fmt.Sprintf("task-pin-first-%d", suffix), "goal", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.CreateTask(fmt.Sprintf("task-pin-second-%d", suffix), "goal", nil, 0, 0)
	if err != nil {
		_ = d.DeleteTask(first.ID)
		t.Fatal(err)
	}
	defer func() {
		_ = d.DeleteTask(first.ID)
		_ = d.DeleteTask(second.ID)
	}()

	pinned := true
	first, err = d.UpdateTask(first.ID, TaskPatch{Pinned: &pinned})
	if err != nil || first == nil || !first.Pinned || first.PinnedAt == nil {
		t.Fatalf("pin first = %+v, %v", first, err)
	}
	firstPinnedAt := *first.PinnedAt
	time.Sleep(5 * time.Millisecond)
	second, err = d.UpdateTask(second.ID, TaskPatch{Pinned: &pinned})
	if err != nil || second == nil || !second.Pinned || second.PinnedAt == nil {
		t.Fatalf("pin second = %+v, %v", second, err)
	}

	name := "renamed pinned task"
	first, err = d.UpdateTask(first.ID, TaskPatch{Name: &name, Pinned: &pinned})
	if err != nil || first == nil {
		t.Fatal(err)
	}
	if first.Name != name || first.PinnedAt == nil || !first.PinnedAt.Equal(firstPinnedAt) {
		t.Fatalf("repeat pin should preserve pin time: %+v (want %v)", first, firstPinnedAt)
	}

	tasks, err := d.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
	positions := map[int64]int{}
	for index, task := range tasks {
		positions[task.ID] = index
	}
	if positions[second.ID] >= positions[first.ID] {
		t.Fatalf("newer pin must sort first: second=%d first=%d", positions[second.ID], positions[first.ID])
	}

	pinned = false
	first, err = d.UpdateTask(first.ID, TaskPatch{Pinned: &pinned})
	if err != nil || first == nil || first.Pinned || first.PinnedAt != nil {
		t.Fatalf("unpin first = %+v, %v", first, err)
	}
}

// 한국어 검증 목적: 일괄 삭제 결과에는 입력 ID 전체가 아니라 실제 존재해 삭제된 대화만 포함되어야 한다.
func TestDeleteConversationsReturnsExistingIDs(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	first, err := d.CreateConversation("mainagent", "batch-delete-first", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.CreateConversation("mainagent", "batch-delete-second", nil)
	if err != nil {
		_ = d.DeleteConversation(first.ID)
		t.Fatal(err)
	}
	defer func() {
		_ = d.DeleteConversation(first.ID)
		_ = d.DeleteConversation(second.ID)
	}()

	missing := int64(1<<62 - 1)
	deleted, err := d.DeleteConversations([]int64{first.ID, missing, second.ID})
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[int64]bool, len(deleted))
	for _, id := range deleted {
		seen[id] = true
	}
	if !seen[first.ID] || !seen[second.ID] || seen[missing] || len(deleted) != 2 {
		t.Fatalf("deleted=%v, want existing ids only", deleted)
	}
}
