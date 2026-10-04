// [한국어 길잡이] goals_test.go의 테스트 읽기
// nil Task를 createGoals에 전달했을 때 모델 호출이나 패닉 없이 nil을 반환하는 작은 경계 테스트다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestCreateGoalsNilTask.
// 이 테스트의 fixture와 단언을 함께 읽으면 일반 경로 외의 예외·경쟁 조건을 이해할 수 있다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"testing"
)

func TestCreateGoalsNilTask(t *testing.T) {
	if got := (&Server{}).createGoals(context.Background(), nil, nil); len(got) != 0 {
		t.Fatalf("nil task produced goals: %+v", got)
	}
}
