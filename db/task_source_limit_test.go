package db

// 한국어 테스트 안내
// 최대 직접 source 수 및 회사 수, 회사 ID 양수 검증·순서 보존 중복 제거를 검사한다.
// 과도한 입력이 트랜잭션을 열기 전에 거절되는지를 확인해 잘못된 요청이 DB 작업을 증폭시키지 않도록 한다.

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// 한국어 검증 목적: 직접 source 입력 개수 제한을 BEGIN 이전에 적용하는지 확인한다.
func TestCreateTaskRejectsTooManySourcesBeforeOpeningTransaction(t *testing.T) {
	sourceIDs := make([]int64, MaxTaskSourceCount+1)
	for i := range sourceIDs {
		sourceIDs[i] = int64(i + 1)
	}

	// No database handle is needed: validation must run before Begin so an
	// oversized request cannot consume a connection or create partial rows.
	_, err := (&DB{}).CreateTaskWithOptions("child", "goal", TaskCreateOptions{SourceTaskIDs: sourceIDs})
	if err == nil || !strings.Contains(err.Error(), "too many source tasks") {
		t.Fatalf("expected source-count validation error, got %v", err)
	}
}

// 한국어 검증 목적: 양수 회사 ID 검증, 최초 순서 보존 중복 제거, 고유 회사 수 상한을 확인한다.
func TestNormalizeTaskCompanyIDs(t *testing.T) {
	got, err := NormalizeTaskCompanyIDs([]int64{4, 2, 4, 7, 2})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int64{4, 2, 7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeTaskCompanyIDs=%v, want %v", got, want)
	}
	if _, err := NormalizeTaskCompanyIDs([]int64{1, 0}); !errors.Is(err, ErrTaskCompanyIDsInvalid) {
		t.Fatalf("invalid company id error=%v", err)
	}
}

// 한국어 검증 목적: 과도한 회사 연결 요청이 DB 트랜잭션을 열기 전에 거절되는지 확인한다.
func TestCreateTaskRejectsTooManyCompaniesBeforeOpeningTransaction(t *testing.T) {
	companyIDs := make([]int64, MaxTaskCompanyCount+1)
	for i := range companyIDs {
		companyIDs[i] = int64(i + 1)
	}

	_, err := (&DB{}).CreateTaskWithOptions("child", "goal", TaskCreateOptions{CompanyIDs: companyIDs})
	if !errors.Is(err, ErrTaskCompanyIDsInvalid) {
		t.Fatalf("expected company-count validation error, got %v", err)
	}
}
