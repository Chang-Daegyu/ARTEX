package db

// 한국어 테스트 안내
// LLM 프로필의 max_tokens가 저장·조회·수정 후 같은 값으로 돌아오는지와 기본값 적용을 검증한다.
// 미설정 기본값과 사용자가 저장한 제한값을 구분해 모델 응답 예산이 의도치 않게 바뀌지 않도록 한다.

import "testing"

// TestProfileMaxTokensRoundTrip pins the output-cap columns through the whole
// read/write surface: both column lists (list vs keyed loads) must carry them,
// and an update must not drop them. A miscounted column list silently shifts
// every later Scan target, so this is the test that catches it.
// 한국어 검증 목적: 모델 응답 토큰 제한을 생성·수정·조회해 같은 값이 돌아오는지 확인한다.
func TestProfileMaxTokensRoundTrip(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	// Close via Cleanup, registered FIRST so it runs LAST: cleanups are LIFO, and a
	// plain `defer d.Close()` would fire before them — the row-deleting cleanups
	// would then run against a closed pool and silently leave test rows behind.
	t.Cleanup(func() { d.Close() })

	id, err := d.SaveProfile(&LLMProfile{
		Name: "t-maxtok", Format: "openai", Model: "m", APIKey: "k",
		MaxTokens: 8192, MaxTokensField: "max_completion_tokens",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM llm_profiles WHERE id=$1`, id) })

	// Keyed single-row load (profileColsKey).
	p, err := d.ProfileByID(id)
	if err != nil || p == nil {
		t.Fatalf("ProfileByID: %v, p=%v", err, p)
	}
	if p.MaxTokens != 8192 || p.MaxTokensField != "max_completion_tokens" {
		t.Fatalf("keyed load: max_tokens=%d field=%q", p.MaxTokens, p.MaxTokensField)
	}

	// List load (profileCols — the hint variant, a separate column list).
	ps, err := d.ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	var found *LLMProfile
	for _, x := range ps {
		if x.ID == id {
			found = x
		}
	}
	if found == nil {
		t.Fatal("profile missing from ListProfiles")
	}
	if found.MaxTokens != 8192 || found.MaxTokensField != "max_completion_tokens" {
		t.Fatalf("list load: max_tokens=%d field=%q", found.MaxTokens, found.MaxTokensField)
	}

	// Update with a blank key takes the "keep existing key" UPDATE branch, which
	// has its own column list and is the easiest one to forget.
	p.APIKey = ""
	p.MaxTokens = 4096
	p.MaxTokensField = ""
	if _, err := d.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	after, err := d.ProfileByID(id)
	if err != nil || after == nil {
		t.Fatalf("reload: %v", err)
	}
	if after.MaxTokens != 4096 || after.MaxTokensField != "" {
		t.Fatalf("after update: max_tokens=%d field=%q", after.MaxTokens, after.MaxTokensField)
	}
	if after.APIKey != "k" {
		t.Fatalf("blank key on update must keep the stored one, got %q", after.APIKey)
	}
}

// A profile saved without touching the new fields must read back as "no cap,
// classic field name" — the pre-feature behaviour old rows also get.
// 한국어 검증 목적: 토큰 제한을 생략했을 때 이전 동작과 호환되는 기본값이 유지되는지 확인한다.
func TestProfileMaxTokensDefaults(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	// Close via Cleanup, registered FIRST so it runs LAST: cleanups are LIFO, and a
	// plain `defer d.Close()` would fire before them — the row-deleting cleanups
	// would then run against a closed pool and silently leave test rows behind.
	t.Cleanup(func() { d.Close() })

	id, err := d.SaveProfile(&LLMProfile{Name: "t-maxtok-default", Format: "openai", Model: "m", APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Exec(`DELETE FROM llm_profiles WHERE id=$1`, id) })

	p, err := d.ProfileByID(id)
	if err != nil || p == nil {
		t.Fatalf("ProfileByID: %v", err)
	}
	if p.MaxTokens != 0 || p.MaxTokensField != "" {
		t.Fatalf("defaults: max_tokens=%d field=%q, want 0 and \"\"", p.MaxTokens, p.MaxTokensField)
	}
}
