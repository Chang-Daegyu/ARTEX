// [한국어 파일 안내] traffic/reclaim_test.go
// 트래픽 삭제가 논리적인 행 제거뿐 아니라 실제 디스크 공간 회수로 이어지는지 검증한다.
// 임시 DB를 충분히 키운 뒤 삭제·배경 정리 완료를 기다리고 파일 크기·freelist·FTS segment를 관찰한다.
// 원시 데이터 전체 삭제와 옛 저장소 형식 전환을 포함하며 finding 증거 저장소와는 별개다.
package traffic

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bulkRecord fills the index with inline bodies — the ones that actually make
// index.sqlite grow. Binary content type keeps them out of the full-text index so
// the test stays fast; the FTS side is covered by TestReclaimMergesFTSTombstones.
// 한국어 해설: 인라인 임계값 아래의 합성 본문을 반복 기록하여 SQLite 본체의 성장과 회수를 측정할 자료를 만든다.
func bulkRecord(tr *Traffic, host string, n, size int) {
	body := []byte(strings.Repeat("A", size))
	for i := 0; i < n; i++ {
		tr.record(newFlow(host, "GET", fmt.Sprintf("/blob/%d", i), nil, body,
			withRespType("application/octet-stream")))
	}
}

// 한국어 해설: 새 인덱스의 플래그와 실제 PRAGMA auto_vacuum 값이 모두 incremental인지 확인한다.
func TestNewIndexEnablesIncrementalVacuum(t *testing.T) {
	tr, _ := openTraffic(t)
	if !tr.incrementalVacuum {
		t.Fatal("新建索引库未启用增量回收")
	}
	var mode int
	if err := tr.DB().QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != autoVacuumIncremental {
		t.Fatalf("auto_vacuum=%d，应为 %d", mode, autoVacuumIncremental)
	}
}

// TestDeleteReclaimsIndexSpace is the regression: deleting traffic used to leave
// index.sqlite at its high-water mark forever, because SQLite only chains freed
// pages onto its freelist and nothing ever returned them to the filesystem.
// 한국어 해설: 대량 인라인 본문 삭제 후 파일 크기가 감소하고 과도한 빈 페이지가 남지 않는지 검증한다.
func TestDeleteReclaimsIndexSpace(t *testing.T) {
	tr, _ := openTraffic(t)
	const host = "bulk.example.com"
	// 30 × 200KB stays under maxInlineBody, so every body lands in the database
	// itself rather than the blob store — that is where the growth was invisible.
	bulkRecord(tr, host, 30, 200*1024)
	grown := tr.indexBytes()
	if grown < 5<<20 {
		t.Fatalf("索引只有 %d 字节，样本不足以验证回收", grown)
	}

	if n, err := tr.DeleteHostsExact([]string{host}); err != nil || n != 30 {
		t.Fatalf("DeleteHostsExact=(%d,%v)，应为 (30,nil)", n, err)
	}
	tr.reaping.Wait() // 回收在后台分块进行

	after := tr.indexBytes()
	if after > grown/4 {
		t.Fatalf("删除后索引仍占 %d 字节（删除前 %d），空间没有还给文件系统", after, grown)
	}
	// A handful of pages incremental_vacuum could not move to the end of the file
	// is a normal residual; the ~1500 that the deletion freed must be gone.
	var free int
	if err := tr.DB().QueryRow(`PRAGMA freelist_count`).Scan(&free); err != nil {
		t.Fatal(err)
	}
	if free > 64 {
		t.Fatalf("仍有 %d 个空闲页未回收", free)
	}
}

// TestReclaimMergesFTSTombstones covers the second half of the leak: ex_fts is a
// contentless_delete index, so a DELETE only writes tombstones. Without a merge
// the index keeps growing on every deletion — deleting traffic made it bigger.
// 한국어 해설: 여러 회차 삭제로 생긴 FTS tombstone들이 합쳐져 빈 인덱스의 작은 구조만 남는지 확인한다.
func TestReclaimMergesFTSTombstones(t *testing.T) {
	tr, _ := openTraffic(t)
	if !tr.fts {
		t.Skip("驱动未启用 FTS5")
	}
	// Deleted in batches, which is what leaves tombstones spread over many
	// segments rather than emptying the index in one shot.
	for round := 0; round < 4; round++ {
		host := fmt.Sprintf("fts%d.example.com", round)
		for i := 0; i < 20; i++ {
			tr.record(newFlow(host, "GET", fmt.Sprintf("/p/%d", i), nil,
				[]byte(strings.Repeat("secret token 中文正文 padding ", 200))))
		}
		if _, err := tr.DeleteHostsExact([]string{host}); err != nil {
			t.Fatal(err)
		}
		tr.reaping.Wait()
	}

	var exchanges, segments int
	if err := tr.DB().QueryRow(`SELECT COUNT(*) FROM exchanges`).Scan(&exchanges); err != nil {
		t.Fatal(err)
	}
	if err := tr.DB().QueryRow(`SELECT COUNT(*) FROM ex_fts_data`).Scan(&segments); err != nil {
		t.Fatal(err)
	}
	if exchanges != 0 {
		t.Fatalf("还剩 %d 条流量", exchanges)
	}
	// A fully merged, empty contentless index keeps only its structure rows.
	if segments > 8 {
		t.Fatalf("全文索引残留 %d 行段数据，tombstone 未被合并回收", segments)
	}
}

// TestReclaimOnLegacyIndexIsHarmless covers installs created before
// auto_vacuum=incremental became the default: incremental_vacuum is a silent
// no-op there, so reclamation must report the situation and finish rather than
// spin or fail. Only a full compaction can convert such a file.
// 한국어 해설: incremental 회수가 지원되지 않는 옛 DB에서도 삭제/회수가 실패하거나 무한 반복하지 않는지 확인한다.
func TestReclaimOnLegacyIndexIsHarmless(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_index"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create the tables first, with auto_vacuum left at its default 0 — exactly the
	// shape Open used to leave behind.
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "_index", "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(indexSchema); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	tr, err := Open(dir, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.Close() })
	if tr.incrementalVacuum {
		t.Fatal("旧库不应报告已启用增量回收")
	}

	const host = "legacy.example.com"
	bulkRecord(tr, host, 8, 200*1024)
	if n, err := tr.DeleteHostsExact([]string{host}); err != nil || n != 8 {
		t.Fatalf("DeleteHostsExact=(%d,%v)，应为 (8,nil)", n, err)
	}
	tr.reaping.Wait() // 必须收敛，不能卡在预算里

	// The freelist stays populated: that is the whole reason a compaction entry
	// point is needed for pre-existing databases.
	var free int
	if err := tr.DB().QueryRow(`PRAGMA freelist_count`).Scan(&free); err != nil {
		t.Fatal(err)
	}
	if free == 0 {
		t.Fatal("旧库居然回收了空闲页，说明测试没有真的构造出旧库")
	}
}

// TestDeleteAllPurgesAndCompacts covers the page's clear-everything action: it
// must leave nothing behind — including host directories the index no longer
// knows about — and it must hand the index space back, since an emptied index is
// the one moment a full rewrite is cheap.
// 한국어 해설: 전체 삭제가 SQL·본문·참조·고아 폴더를 정리하고 VACUUM 뒤에도 새 기록이 가능한지 검증한다.
func TestDeleteAllPurgesAndCompacts(t *testing.T) {
	tr, dir := openTraffic(t)
	bulkRecord(tr, "a.example.com", 10, 200*1024)
	bulkRecord(tr, "b.example.com", 10, 200*1024)
	// A text body so the full-text index has real content, and a spilled one so a
	// blob exists to collect.
	tr.record(newFlow("c.example.com", "GET", "/page", nil, []byte(strings.Repeat("secret-token ", 500))))
	tr.record(newFlow("c.example.com", "GET", "/big", nil,
		[]byte(strings.Repeat("B", maxInlineBody+1024)), withRespType("application/sql")))
	// An orphaned legacy directory: no index row points at it, so only a
	// clear-everything should take it.
	orphan := filepath.Join(dir, "orphan.example.com")
	if err := os.MkdirAll(orphan, 0o755); err != nil {
		t.Fatal(err)
	}
	grown := tr.indexBytes()
	if grown < 5<<20 {
		t.Fatalf("索引只有 %d 字节，样本不足", grown)
	}

	deleted, reclaimed, err := tr.DeleteAll()
	if err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	if deleted != 22 {
		t.Fatalf("deleted=%d，应为 22", deleted)
	}
	tr.reaping.Wait()

	if reclaimed < grown/2 {
		t.Fatalf("只回收了 %d 字节（删除前索引 %d）", reclaimed, grown)
	}
	if after := tr.indexBytes(); after > grown/8 {
		t.Fatalf("清空后索引仍占 %d 字节（删除前 %d）", after, grown)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM exchanges`,
		`SELECT COUNT(*) FROM exchange_bodies`,
		`SELECT COUNT(*) FROM blob_refs`,
	} {
		var c int
		if err := tr.DB().QueryRow(q).Scan(&c); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Fatalf("%s = %d，应为 0", q, c)
		}
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("孤立的历史 host 目录未被清理：%v", err)
	}
	// Recording must keep working against the freshly rewritten file.
	tr.record(newFlow("d.example.com", "GET", "/after", nil, []byte("清空后仍可录制")))
	if n, err := tr.Count(); err != nil || n != 1 {
		t.Fatalf("清空后 Count=(%d,%v)，应为 (1,nil)", n, err)
	}
}

// TestDeleteAllConvertsLegacyIndex is why the purge compacts rather than just
// deleting: auto_vacuum cannot be switched on after the fact except through a
// VACUUM, and an emptied index is the cheapest place to pay for one. After this,
// ordinary deletions reclaim space on their own.
// 한국어 해설: 전체 삭제가 옛 auto_vacuum=0 DB를 전환하여 이후 일반 호스트 삭제에도 공간 회수가 적용되는지 확인한다.
func TestDeleteAllConvertsLegacyIndex(t *testing.T) {
	dir := t.TempDir()
	old := openLegacyIndex(t, dir)
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	tr, err := Open(dir, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.Close() })
	if tr.incrementalVacuum {
		t.Fatal("旧库不应报告已启用增量回收")
	}

	bulkRecord(tr, "legacy.example.com", 10, 200*1024)
	if _, _, err := tr.DeleteAll(); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	if !tr.incrementalVacuum {
		t.Fatal("清空后旧库未被转换为增量回收模式")
	}

	// The converted database now reclaims on an ordinary host deletion.
	bulkRecord(tr, "again.example.com", 10, 200*1024)
	grown := tr.indexBytes()
	if _, err := tr.DeleteHostsExact([]string{"again.example.com"}); err != nil {
		t.Fatal(err)
	}
	tr.reaping.Wait()
	if after := tr.indexBytes(); after > grown/4 {
		t.Fatalf("转换后普通删除仍未回收：%d 字节（删除前 %d）", after, grown)
	}
}
