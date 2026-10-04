// Package report renders a deliverable Markdown report from the exploration
// graph: confirmed findings (with severity/evidence) (docs §15 P5).
// [한국어 파일 안내] report/report.go
// 탐색 그래프의 finding 노드를 가벼운 Markdown 보고서로 렌더링한다.
// 입력 데이터의 유효성·증거의 진위 판단은 호출자가 맡고 이 파일은 표시·집계·순서만 담당한다.
// 보고서의 중국어 제목과 라벨은 실행 출력이므로 주석 한국어화에서 그대로 보존했다.
package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Autumn-27/artex/db"
)

// Input bundles what the report needs.
// 한국어 자료형: 보고서 렌더링에 필요한 이미 조회된 작업 정보·자산 집계·finding 노드 묶음이다.
type Input struct {
	Title       string
	Goal        string
	GeneratedAt time.Time
	AssetCounts map[string]int
	Findings    []*db.Node
}

// 한국어 자료형: 그래프 JSON 중 보고서에 필요한 표시용 필드만 남긴 내부 구조체다.
type findingView struct {
	VulnClass string
	Name      string
	Severity  string
	Summary   string
	PoC       string
}

// 한국어 해설: 그래프 노드의 JSON payload에서 이름·심각도·요약·PoC를 표시 구조로 뽑는다. 파싱 오류는 현재 구현에서 빈 필드로 남는다.
func parseFinding(n *db.Node) findingView {
	var p struct {
		VulnClass string `json:"vulnclass"`
		Name      string `json:"name"`
		Severity  string `json:"severity"`
		Summary   string `json:"summary"`
		Evidence  struct {
			PoC string `json:"poc"`
		} `json:"evidence"`
	}
	_ = json.Unmarshal(n.Payload, &p)
	return findingView{p.VulnClass, p.Name, p.Severity, p.Summary, p.Evidence.PoC}
}

var sevRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3, "": 4}

// Markdown renders the report.
// 한국어 해설: 목표·생성 시각·자산 수와 finding 목록을 Markdown으로 만든다. 자산 종류는 이름순, 발견은 심각도 순으로 배치한다.
func Markdown(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 渗透测试报告 — %s\n\n", nz(in.Title, "未命名任务"))
	fmt.Fprintf(&b, "- **任务目标**：%s\n", nz(in.Goal, "（未指定）"))
	fmt.Fprintf(&b, "- **生成时间**：%s\n\n", in.GeneratedAt.Format("2006-01-02 15:04:05"))

	// summary
	fmt.Fprintf(&b, "## 摘要\n\n")
	fmt.Fprintf(&b, "- 确认发现：**%d** 个\n", len(in.Findings))
	fmt.Fprintf(&b, "- 资产：")
	var types []string
	for t := range in.AssetCounts {
		types = append(types, t)
	}
	sort.Strings(types)
	for i, t := range types {
		if i > 0 {
			b.WriteString("、")
		}
		fmt.Fprintf(&b, "%s %d", t, in.AssetCounts[t])
	}
	b.WriteString("\n\n")

	// findings
	fmt.Fprintf(&b, "## 发现\n\n")
	if len(in.Findings) == 0 {
		b.WriteString("_本次未确认漏洞。_\n\n")
	} else {
		fs := make([]findingView, 0, len(in.Findings))
		for _, n := range in.Findings {
			fs = append(fs, parseFinding(n))
		}
		sort.SliceStable(fs, func(i, j int) bool { return sevRank[fs[i].Severity] < sevRank[fs[j].Severity] })
		for i, f := range fs {
			fmt.Fprintf(&b, "### %d. [%s] %s\n\n", i+1, strings.ToUpper(nz(f.Severity, "info")), nz(f.Name, nz(f.VulnClass, "未分类")))
			fmt.Fprintf(&b, "%s\n\n", nz(f.Summary, ""))
			if f.PoC != "" {
				fmt.Fprintf(&b, "**PoC / 证据：**\n\n```\n%s\n```\n\n", f.PoC)
			}
		}
	}

	return b.String()
}

// 한국어 해설: 공백뿐인 표시 문자열을 호출자가 지정한 기본 문구로 바꾼다.
func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
