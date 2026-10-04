package db

// 한국어 테스트 안내
// 회사 범위 변경용 advisory lock 키가 스키마/테스트/증거 등 다른 기반 잠금과 충돌하지 않는지 검사한다.
// 같은 정수 키를 재사용하면 논리적으로 관계없는 작업들이 서로 대기하게 되므로 이름만 다른 잠금으로는 충분하지 않다.

import "testing"

// 한국어 검증 목적: 회사 범위 잠금의 정수 키가 다른 기반 잠금 키와 충돌하지 않는지 확인한다.
func TestCompanyScopeMutationLockDoesNotReuseInfrastructureLocks(t *testing.T) {
	reserved := map[string]int64{
		"schema migration":         7337741001,
		"cross-package test suite": 7337741002,
	}
	for name, key := range reserved {
		if companyScopeMutationLock == key {
			t.Fatalf("company scope mutation lock reuses the %s advisory lock key %d", name, key)
		}
	}
}
