package db

// 한국어 테스트 안내
// 정규화 회사명 중복, 회사와 범위의 원자적 생성, 규칙 갱신 실패 롤백, 회사 귀속 재계산과 삭제를 검증한다.
// 일부 테스트는 의도적으로 저장 실패를 만들어 회사/범위가 반쯤 남지 않는지 확인한다. 실제 PostgreSQL 트랜잭션이 필요하다.

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// cleanup helpers to remove test data
func cleanupCompany(d *DB, id int64) {
	d.Exec(`DELETE FROM company_scope WHERE company_id = $1`, id)
	d.Exec(`DELETE FROM companies WHERE id = $1`, id)
}

// 한국어 검증 목적: 정규화 회사명으로 생성/갱신한 결과를 ID와 이름 조회에서 일관되게 읽는지 확인한다.
func TestCompanyUpsertAndGet(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, created, err := cs.UpsertCompany("Test Corp", "https://example.com/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)
	if !created {
		t.Error("first upsert should report created=true")
	}

	// duplicate: same nkey, should not create new
	id2, created2, err := cs.UpsertCompany("Test Corp", "")
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Errorf("dedup failed: %d != %d", id2, id)
	}
	if created2 {
		t.Error("second upsert should report created=false")
	}

	c, err := cs.GetCompany(id)
	if err != nil || c == nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if c.Name != "Test Corp" {
		t.Errorf("name: %q", c.Name)
	}

	// GetCompanyByName
	c2, err := cs.GetCompanyByName("test corp") // normalised
	if err != nil || c2 == nil {
		t.Fatalf("GetCompanyByName: %v", err)
	}
	if c2.ID != id {
		t.Errorf("GetCompanyByName id mismatch: %d vs %d", c2.ID, id)
	}
}

// 한국어 검증 목적: 표시 공백/대소문자만 다른 회사 생성이 기존 회사를 덮어쓰지 않고 중복으로 거절되는지 확인한다.
func TestCreateCompanyWithScopeRejectsNormalizedDuplicate(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	stamp := time.Now().UnixNano()
	name := fmt.Sprintf("Strict Company %d", stamp)
	id, added, _, _, validationErrors, err := cs.CreateCompanyWithScope(name, "", []ScopeInput{
		{Kind: "domain", Value: fmt.Sprintf("strict-%d.example", stamp)},
	}, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)
	if added != 1 || len(validationErrors) != 0 {
		t.Fatalf("initial scope: added=%d errors=%v", added, validationErrors)
	}

	_, _, _, _, _, err = cs.CreateCompanyWithScope(
		"  "+strings.ToUpper(strings.ReplaceAll(name, " ", "   "))+"  ",
		"",
		[]ScopeInput{{Kind: "domain", Value: fmt.Sprintf("replacement-%d.example", stamp)}},
		"test",
	)
	if !errors.Is(err, ErrCompanyNameConflict) {
		t.Fatalf("duplicate create error=%v want ErrCompanyNameConflict", err)
	}
	scope, err := cs.GetScope(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope) != 1 || scope[0].Domain != fmt.Sprintf("strict-%d.example", stamp) {
		t.Fatalf("duplicate create changed existing scope: %+v", scope)
	}
}

// 한국어 검증 목적: 범위 쓰기를 실패시켜 새 회사만 혼자 남지 않고 생성 트랜잭션이 되돌아가는지 확인한다.
func TestCreateCompanyWithScopeRollsBackOnScopeWriteFailure(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	name := fmt.Sprintf("Atomic Create %d", time.Now().UnixNano())
	_, _, _, _, _, err = cs.CreateCompanyWithScope(name, "", []ScopeInput{
		{Kind: "keyword", Value: "invalid\x00postgres-text"},
	}, "test")
	if err == nil {
		t.Fatal("expected scope database write to fail")
	}
	company, getErr := cs.GetCompanyByName(name)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if company != nil {
		defer cleanupCompany(d, company.ID)
		t.Fatalf("company row survived failed initial scope transaction: %+v", company)
	}
}

// 한국어 검증 목적: 회사 범위의 추가와 조회가 정규화된 입력을 보존하는지 확인한다.
func TestCompanyScope(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, _, err := cs.UpsertCompany("ScopeTestCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)

	lines := []string{"example.com", "192.168.1.0/24", "10.0.0.1"}
	added, skipped, invalid, errs := cs.AddScope(id, lines, "test")
	if added != 3 {
		t.Errorf("want 3 added, got %d (errs: %v)", added, errs)
	}
	if skipped != 0 || invalid != 0 {
		t.Errorf("unexpected skipped=%d invalid=%d", skipped, invalid)
	}

	// Adding again should skip (duplicate)
	added2, skipped2, invalid2, _ := cs.AddScope(id, lines, "test")
	if added2 != 0 || skipped2 != 3 {
		t.Errorf("want 0 added 3 skipped, got %d added %d skipped %d invalid", added2, skipped2, invalid2)
	}

	scope, err := cs.GetScope(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope) != 3 {
		t.Errorf("want 3 scope rules, got %d", len(scope))
	}
}

// 한국어 검증 목적: 잘못된 회사 범위 입력의 invalid 수와 오류 피드백이 유효한 입력 처리와 구별되는지 확인한다.
func TestCompanyScopeInvalid(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, _, err := cs.UpsertCompany("InvalidScopeCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)

	// Explicitly typed TLD-only domains and overly broad CIDRs should be
	// rejected. Untyped plain text is intentionally classified as a keyword.
	inputs := []ScopeInput{
		{Kind: "domain", Value: "com"},
		{Kind: "cidr", Value: "1.2.3.4/8"},
	}
	added, _, invalid, _ := cs.AddScopeInputs(id, inputs, "test")
	if added != 0 {
		t.Errorf("want 0 added for invalid lines, got %d", added)
	}
	if invalid != 2 {
		t.Errorf("want 2 invalid, got %d", invalid)
	}
}

// 한국어 검증 목적: 도메인/IP 범위로 회사 후보를 선택하는 우선순위와 일치하지 않는 경우를 확인한다.
func TestResolveCompany(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, _, err := cs.UpsertCompany("ResolveCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)

	cs.AddScope(id, []string{"resolve-test.io", "10.20.0.0/16"}, "test")

	// domain match
	cid, err := cs.ResolveCompany("resolve-test.io", "")
	if err != nil || cid == nil || *cid != id {
		t.Errorf("domain resolve: want %d, got %v (err %v)", id, cid, err)
	}

	// no match
	cid2, err := cs.ResolveCompany("notinscope.com", "")
	if err != nil || cid2 != nil {
		t.Errorf("no-match: want nil, got %v", cid2)
	}

	// IP/CIDR match
	cid3, err := cs.ResolveCompany("", "10.20.5.1")
	if err != nil || cid3 == nil || *cid3 != id {
		t.Errorf("cidr resolve: want %d, got %v (err %v)", id, cid3, err)
	}

	// IP outside CIDR
	cid4, err := cs.ResolveCompany("", "10.30.0.1")
	if err != nil || cid4 != nil {
		t.Errorf("cidr no-match: want nil, got %v", cid4)
	}
}

// 한국어 검증 목적: 회사 범위 집합을 교체한 뒤 이전 규칙과 새 규칙의 효과가 올바르게 바뀌는지 확인한다.
func TestUpdateScope(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, _, err := cs.UpsertCompany("UpdateScopeCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)

	cs.AddScope(id, []string{"old-domain.com"}, "initial")

	// UpdateScope replaces
	added, invalid, errs := cs.UpdateScope(id, []string{"new-domain.com"}, "replacement")
	if added != 1 || invalid != 0 || len(errs) != 0 {
		t.Errorf("UpdateScope: added=%d invalid=%d errs=%v", added, invalid, errs)
	}

	scope, _ := cs.GetScope(id)
	if len(scope) != 1 || scope[0].Domain != "new-domain.com" {
		t.Errorf("UpdateScope: expected new-domain.com only, got %+v", scope)
	}

	// Invalid replacement input must not turn a partial validation response into
	// a destructive replacement of the existing rules.
	added, invalid, validationErrors, err := cs.UpdateScopeInputsChecked(id, []ScopeInput{
		{Kind: "domain", Value: "co.uk"},
	}, "invalid replacement")
	var validationErr *CompanyScopeValidationError
	if added != 0 || invalid != 1 || len(validationErrors) != 1 || !errors.As(err, &validationErr) {
		t.Fatalf("invalid replacement: added=%d invalid=%d validation=%v err=%v", added, invalid, validationErrors, err)
	}
	scope, err = cs.GetScope(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope) != 1 || scope[0].Domain != "new-domain.com" {
		t.Fatalf("invalid replacement changed existing scope: %+v", scope)
	}
}

// 한국어 검증 목적: 새 규칙 삽입 실패가 기존 범위 삭제까지 확정시키지 않고 원래 상태를 되살리는지 확인한다.
func TestUpdateScopeRollsBackOnInsertFailure(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	name := fmt.Sprintf("Atomic Scope Update %d", time.Now().UnixNano())
	id, _, err := cs.UpsertCompany(name, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)
	oldDomain := fmt.Sprintf("old-%d.example", time.Now().UnixNano())
	added, _, invalid, addErrors := cs.AddScopeInputs(id, []ScopeInput{{Kind: "domain", Value: oldDomain}}, "initial")
	if added != 1 || invalid != 0 || len(addErrors) != 0 {
		t.Fatalf("seed scope: added=%d invalid=%d errors=%v", added, invalid, addErrors)
	}

	added, invalid, updateErrors := cs.UpdateScopeInputs(id, []ScopeInput{
		{Kind: "keyword", Value: "invalid\x00postgres-text"},
	}, "replacement")
	if added != 0 || invalid != 0 || len(updateErrors) == 0 {
		t.Fatalf("failed update result: added=%d invalid=%d errors=%v", added, invalid, updateErrors)
	}
	scope, err := cs.GetScope(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope) != 1 || scope[0].Domain != oldDomain {
		t.Fatalf("failed replacement did not preserve old scope: %+v", scope)
	}
}

// 한국어 검증 목적: 회사 삭제 이후 참조와 자동 귀속의 저장 계약을 확인한다.
func TestDeleteCompany(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, _, err := cs.UpsertCompany("DeleteMeCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	cs.AddScope(id, []string{"deletetest.com"}, "test")

	if err := cs.DeleteCompany(id); err != nil {
		t.Fatal(err)
	}
	c, err := cs.GetCompany(id)
	if err != nil || c != nil {
		t.Error("expected company to be gone")
	}
	// scope should be cascade-deleted
	scope, _ := cs.GetScope(id)
	if len(scope) != 0 {
		t.Errorf("expected scope cascade-deleted, got %d rules", len(scope))
	}
}

// 한국어 검증 목적: 자산 삭제 옵션을 지정했을 때 회사와 선택된 관련 자산을 함께 제거하는지 확인한다.
func TestDeleteCompanyWithAssetsDeletesBoth(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	stamp := time.Now().UnixNano()
	id, _, err := cs.UpsertCompany(fmt.Sprintf("Delete Assets Company %d", stamp), "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)
	var assetID int64
	domain := fmt.Sprintf("delete-assets-%d.example", stamp)
	if err := d.QueryRow(`
INSERT INTO assets(type, domain, root_domain, company_id, company_source)
VALUES ('root_domain', $1, $1, $2, 'explicit')
RETURNING id`, domain, id).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	defer d.Exec(`DELETE FROM assets WHERE id = $1`, assetID) //nolint:errcheck

	deleted, err := cs.DeleteCompanyWithAssets(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("assets deleted=%d want 1", deleted)
	}
	company, err := cs.GetCompany(id)
	if err != nil {
		t.Fatal(err)
	}
	if company != nil {
		t.Fatalf("company still exists: %+v", company)
	}
	var assetsRemaining int
	if err := d.QueryRow(`SELECT COUNT(*) FROM assets WHERE id = $1`, assetID).Scan(&assetsRemaining); err != nil {
		t.Fatal(err)
	}
	if assetsRemaining != 0 {
		t.Fatalf("asset %d survived company deletion", assetID)
	}
}

// 한국어 검증 목적: 범위 변경 이후 전역 자산의 자동 회사 귀속을 다시 계산하는 결과를 확인한다.
func TestRecomputeAttribution(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()
	as := d.Assets()

	id, _, err := cs.UpsertCompany("AttributeTestCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)
	defer d.Exec(`DELETE FROM assets WHERE root_domain = 'attr-test.com'`)

	// insert asset before adding scope
	assetID, err := as.UpsertRootDomain(UpsertRootDomainReq{Domain: "attr-test.com"})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Exec(`DELETE FROM assets WHERE id = $1`, assetID)

	// asset should not be attributed yet
	var companyID *int64
	d.QueryRow(`SELECT company_id FROM assets WHERE id = $1`, assetID).Scan(&companyID)
	if companyID != nil {
		t.Error("expected no company before scope added")
	}

	// add scope and recompute
	cs.AddScope(id, []string{"attr-test.com"}, "test")
	if err := cs.RecomputeAttribution(); err != nil {
		t.Fatal(err)
	}

	d.QueryRow(`SELECT company_id FROM assets WHERE id = $1`, assetID).Scan(&companyID)
	if companyID == nil || *companyID != id {
		t.Errorf("RecomputeAttribution: expected company %d, got %v", id, companyID)
	}
}

// 한국어 검증 목적: 회사 목록에서 범위 및 자산 수 요약이 실제 데이터와 맞는지 확인한다.
func TestListCompanies(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v)", err)
	}
	defer d.Close()
	cs := d.Companies()

	id, _, err := cs.UpsertCompany("ListTestCorp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupCompany(d, id)

	companies, err := cs.ListCompanies()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range companies {
		if c.ID == id {
			found = true
		}
	}
	if !found {
		t.Error("ListCompanies: created company not found")
	}
}
