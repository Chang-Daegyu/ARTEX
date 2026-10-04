// [한국어 파일 안내] notify/render_test.go
// 메시지 절단의 모든 바이트/문자 경계와 HTML 끝부분을 직접 검사한다.
// 다중 바이트 문자·이모지·엔티티를 잘못 자르면 전체 알림 거절로 이어지므로 형식의 유효성과 제한 길이를 함께 확인한다.
package notify

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 한국어 해설: 여러 혼합 문자열의 모든 절단 위치에서 UTF-8이 유효하고 바이트 예산을 넘지 않는지 확인한다.
func TestTruncateBytesKeepsValidUTF8(t *testing.T) {
	// 这是本包最要紧的一条不变量。企微按**字节**限长，中文 3 字节/字，
	// 任何按字节硬切的实现都会把汉字切成半个，产出非法 UTF-8 而被平台拒收。
	// 用长度互质的多种中英混排输入去撞每一个可能的切点。
	inputs := []string{
		"中文测试内容",
		"混合 mixed 内容 content",
		"a中b文c测d试e",
		"🔴🟠🟡🔵", // 4 字节 emoji，切错更明显
		strings.Repeat("漏洞", 100),
	}
	for _, in := range inputs {
		for max := 1; max <= len(in)+2; max++ {
			got := TruncateBytes(in, max)
			if !utf8.ValidString(got) {
				t.Fatalf("输入 %q max=%d: 产出非法 UTF-8 %q", in, max, got)
			}
			if len(got) > max {
				t.Fatalf("输入 %q max=%d: 结果 %d 字节超出上限", in, max, len(got))
			}
			// 未被截断时不得改动内容。
			if len(in) <= max && got != in {
				t.Fatalf("输入 %q max=%d: 未超限却改动了内容 -> %q", in, max, got)
			}
		}
	}
}

// 한국어 해설: 0 또는 음수 한도는 문자열을 제한하지 않는다는 공통 계약을 검증한다.
func TestTruncateBytesZeroMeansUnlimited(t *testing.T) {
	long := strings.Repeat("x", 10000)
	if got := TruncateBytes(long, 0); got != long {
		t.Fatal("max=0 应表示不限制")
	}
	if got := TruncateBytes(long, -5); got != long {
		t.Fatal("max<0 应表示不限制")
	}
}

// 한국어 해설: 생략 부호보다 작은 예산에도 결과가 제한을 넘지 않고 일반 예산에서는 부호를 붙이는지 확인한다.
func TestTruncateBytesEllipsisBudget(t *testing.T) {
	// max 小于省略号本身时，不能因为追加省略号而反过来超限。
	got := TruncateBytes("abcdefgh", 1)
	if len(got) > 1 {
		t.Fatalf("max=1 时结果 %q 长度 %d 超限", got, len(got))
	}
	// 正常情况应带省略号。
	if got := TruncateBytes("abcdefgh", 5); !strings.HasSuffix(got, ellipsis) {
		t.Fatalf("期望带省略号，得到 %q", got)
	}
}

// 한국어 해설: 같은 다중 바이트 문자열에서 rune 한도와 바이트 한도의 결과가 의도대로 다른지 확인한다.
func TestTruncateRunesCountsCharactersNotBytes(t *testing.T) {
	// 与 TruncateBytes 的口径差异必须保留：Telegram 按字符限长，
	// 用字节口径会把中文消息切到只剩三分之一。
	s := "一二三四五六七八九十"
	got := TruncateRunes(s, 5)
	if n := utf8.RuneCountInString(got); n != 5 {
		t.Fatalf("期望 5 个字符，得到 %d 个 (%q)", n, got)
	}
	// 同样的字符串按字节口径应明显更短。
	if utf8.RuneCountInString(TruncateBytes(s, 5)) >= 5 {
		t.Fatal("字节口径不应产出与字符口径相同的字符数")
	}
}

// 한국어 해설: 줄바꿈·탭·중복 공백을 하나의 공백으로 합치고 이후 문자 한도도 유지하는지 검사한다.
func TestOneLineCollapsesWhitespace(t *testing.T) {
	got := OneLine("第一行\n\n第二行\t带制表   多空格", 0)
	if strings.ContainsAny(got, "\n\t") {
		t.Fatalf("应折叠所有空白，得到 %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Fatalf("不应保留连续空格，得到 %q", got)
	}
	// 截断后仍须可读且合法。
	got = OneLine("一二三四五六七八九十", 4)
	if n := utf8.RuneCountInString(got); n != 4 {
		t.Fatalf("期望 4 字符，得到 %d (%q)", n, got)
	}
}

// 한국어 해설: 가능한 모든 문자 절단 위치에서 마지막 HTML 태그 조각을 남기지 않는지 확인한다.
func TestTruncateHTMLNeverCutsTagInHalf(t *testing.T) {
	// 直接截断 HTML 会切出 `<a href="htt` 这种残片，平台会拒收整条消息。
	s := `<b>标题</b>正文正文正文<a href="https://example.com/very/long/path">查看详情</a>`
	for max := 1; max <= utf8.RuneCountInString(s)+2; max++ {
		got := TruncateHTML(s, max)
		if n := utf8.RuneCountInString(got); max > 0 && n > max {
			t.Fatalf("max=%d: 结果 %d 字符超限", max, n)
		}
		// 尾部不能有未闭合的 `<`（即最后一段里出现 `<` 却无 `>`）。
		if lt := strings.LastIndex(got, "<"); lt >= 0 && !strings.Contains(got[lt:], ">") {
			t.Fatalf("max=%d: 尾部标签被切断 -> %q", max, got)
		}
	}
}

// 한국어 해설: 자산이 없을 때·전부 들어갈 때·일부만 보일 때의 표시와 전체 개수 안내를 확인한다.
func TestAssetLineOmitsExcess(t *testing.T) {
	if got := assetLine(nil, 3); got != "" {
		t.Fatalf("无资产应返回空串，得到 %q", got)
	}
	if got := assetLine([]string{"a", "b"}, 3); got != "a、b" {
		t.Fatalf("未超限应全列，得到 %q", got)
	}
	// 超出上限时必须标注总数，否则读者不知道还有多少资产没列出来。
	got := assetLine([]string{"a", "b", "c", "d", "e"}, 2)
	if !strings.Contains(got, "等 5 个") {
		t.Fatalf("应标注总数 5，得到 %q", got)
	}
}

// 한국어 해설: 등급 문턱과 알려진 상태 라벨, 알 수 없는 상태의 원문 보존을 검증한다.
func TestSeverityAndStatusLabels(t *testing.T) {
	if AtLeast("", "low") {
		t.Fatal("空级别序数为 0，应被任何门槛挡住")
	}
	if !AtLeast("critical", "") {
		t.Fatal("空门槛应放行")
	}
	if got := StatusLabel("fixed"); got != "已修复" {
		t.Fatalf("未知状态映射，得到 %q", got)
	}
	// 未知状态原样回显，不臆造标签。
	if got := StatusLabel("weird_status"); got != "weird_status" {
		t.Fatalf("未知状态应原样回显，得到 %q", got)
	}
}

// TestTruncateHTMLNeverCutsEntity 覆盖审计指出的一处遗漏：截断不只要避开
// 半截标签，还要避开被切断的 HTML 实体。
//
// `&amp;` 被切成 `&amp` 之后，一个只认实体的解析器可能拒收**整条**消息——
// 而超长汇总消息本来就常见，代价太大。
// 한국어 해설: &amp; 같은 엔티티를 잘랐을 때 ;가 없는 반쪽 표현이 끝에 남지 않는지 확인한다.
func TestTruncateHTMLNeverCutsEntity(t *testing.T) {
	s := "aaaa&amp;bbbb&lt;cccc&quot;dddd"
	for max := 1; max <= utf8.RuneCountInString(s)+2; max++ {
		got := TruncateHTML(s, max)
		// 尾部不得出现「有 & 但没有对应 ;」的实体残片。
		if amp := strings.LastIndex(got, "&"); amp >= 0 && !strings.Contains(got[amp:], ";") {
			t.Fatalf("max=%d: 尾部留下实体残片 %q", max, got[amp:])
		}
		if strings.Contains(got, "&amp\x00") {
			t.Fatalf("max=%d: 出现畸形实体", max)
		}
	}
}
