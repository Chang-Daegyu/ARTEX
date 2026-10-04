package db

// 한국어 테스트 안내
// UTF-8 문자열을 바이트 제한에 맞춰 자를 때 한글 같은 다중 바이트 문자가 중간에서 깨지지 않는지 검증한다.
// 처음부터 잘못된 UTF-8 입력도 복구하는 경계 조건을 검사하며 DB 연결이 필요하지 않다.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 한국어 검증 목적: 한글 등 다중 바이트 문자열을 길이 제한으로 잘라도 문자 경계가 깨지지 않는지 확인한다.
func TestTruncateUTF8PreservesValidEncoding(t *testing.T) {
	t.Parallel()
	input := strings.Repeat("额度不足", 400)
	got := truncateUTF8(input, 1000)
	if len(got) > 1000 {
		t.Fatalf("truncated value has %d bytes, want at most 1000", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("truncated value is not valid UTF-8: %q", got[len(got)-8:])
	}
}

// 한국어 검증 목적: 원래 입력의 잘못된 UTF-8 바이트도 복구한 뒤 길이 제한을 적용하는지 확인한다.
func TestTruncateUTF8RepairsInvalidInput(t *testing.T) {
	t.Parallel()
	got := truncateUTF8("bad\xffvalue", 1000)
	if !utf8.ValidString(got) {
		t.Fatalf("repaired value is not valid UTF-8: %q", got)
	}
}
