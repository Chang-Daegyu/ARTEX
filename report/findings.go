// [한국어 파일 안내] report/findings.go
// findings 테이블의 발견을 일괄/단일 Markdown과 CSV로 내보내는 렌더러다.
// 증거 버전과 보고서 작성 시 증거 버전이 다르면 오래된 보고서임을 표시한다.
// HTTP 본문 첨부 파일 자체의 복사는 evidence/server가 맡고 여기서는 첨부 링크와 메타데이터를 만든다.
package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Autumn-27/artex/db"
)

// 发现页「导出」用的渲染:把一批 findings 表行渲染成汇总 Markdown、单条 Markdown、
// 或 CSV。JSON 由 server 层直接用 DTO 序列化,不在此处。

// sortFindingsForExport 按严重等级降序、再按时间倒序排,与汇总报告的分组一致。
// 한국어 해설: 심각도가 높은 항목부터, 같은 등급이면 최신 발견부터 정렬한다. 전달 슬라이스 자체의 순서를 바꾼다.
func sortFindingsForExport(fs []*db.DBFinding) {
	sort.SliceStable(fs, func(i, j int) bool {
		ri, rj := sevRank[fs[i].Severity], sevRank[fs[j].Severity]
		if ri != rj {
			return ri < rj // sevRank 越小越严重
		}
		return fs[i].CreatedAt.After(fs[j].CreatedAt)
	})
}

// findingTitle 取漏洞可读标题:名称 → 类别 → 「未分类」。
// 한국어 해설: 발견 이름 → 취약점 분류 → 기본 문구 순서로 표시 제목을 결정한다.
func findingTitle(f *db.DBFinding) string {
	return nz(f.Name, nz(f.VulnClass, "未分类"))
}

// FindingsMarkdown 把一批 findings 整合成一份汇总报告(摘要 + 按严重等级分组,
// 每条含类别/状态/所属任务/证据/详细报告)。
// 한국어 해설: 입력 슬라이스를 복사해 정렬한 후 등급별 건수와 발견의 요약·증거·상세 보고서를 합친다.
func FindingsMarkdown(fs []*db.DBFinding, generatedAt time.Time) string {
	items := append([]*db.DBFinding(nil), fs...)
	sortFindingsForExport(items)

	var b strings.Builder
	b.WriteString("# 漏洞发现汇总报告\n\n")
	fmt.Fprintf(&b, "- **生成时间**：%s\n", generatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- **发现总数**：%d 个\n\n", len(items))

	// 摘要:各严重等级计数。
	counts := map[string]int{}
	for _, f := range items {
		counts[f.Severity]++
	}
	b.WriteString("## 摘要\n\n")
	b.WriteString("| 严重等级 | 数量 |\n| --- | --- |\n")
	for _, s := range []struct{ key, label string }{
		{"critical", "严重"}, {"high", "高危"}, {"medium", "中危"}, {"low", "低危"},
	} {
		fmt.Fprintf(&b, "| %s | %d |\n", s.label, counts[s.key])
	}
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString("_无匹配的漏洞。_\n")
		return b.String()
	}

	b.WriteString("## 漏洞明细\n\n")
	for i, f := range items {
		fmt.Fprintf(&b, "### %d. [%s] %s\n\n", i+1, strings.ToUpper(nz(f.Severity, "info")), findingTitle(f))
		if f.VulnClass != "" {
			fmt.Fprintf(&b, "- **类别**：%s\n", f.VulnClass)
		}
		fmt.Fprintf(&b, "- **状态**：%s\n", nz(f.Status, "pending"))
		if desc := strings.TrimSpace(f.TaskDescription); desc != "" {
			fmt.Fprintf(&b, "- **所属任务**：%s\n", desc)
		}
		fmt.Fprintf(&b, "- **发现时间**：%s\n\n", f.CreatedAt.Format("2006-01-02 15:04:05"))
		if s := strings.TrimSpace(f.Summary); s != "" {
			fmt.Fprintf(&b, "%s\n\n", s)
		}
		if e := strings.TrimSpace(f.Evidence); e != "" {
			fmt.Fprintf(&b, "**证据：**\n\n```\n%s\n```\n\n", e)
		}
		if rep := strings.TrimSpace(f.Report); rep != "" {
			b.WriteString("**详细报告：**\n\n")
			b.WriteString(rep)
			b.WriteString("\n\n")
		}
		b.WriteString(findingTrafficMarkdown(f, false))
		b.WriteString("---\n\n")
	}
	return b.String()
}

// SingleFindingMarkdown 渲染单条漏洞为一份独立 Markdown(用于「一漏洞一文件」打包)。
// 한국어 해설: ZIP의 발견별 독립 Markdown에 들어갈 메타데이터와 본문·첨부 증거 링크를 작성한다.
func SingleFindingMarkdown(f *db.DBFinding, generatedAt time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# [%s] %s\n\n", strings.ToUpper(nz(f.Severity, "info")), findingTitle(f))
	if f.VulnClass != "" {
		fmt.Fprintf(&b, "- **类别**：%s\n", f.VulnClass)
	}
	fmt.Fprintf(&b, "- **严重等级**：%s\n", nz(f.Severity, "info"))
	fmt.Fprintf(&b, "- **状态**：%s\n", nz(f.Status, "pending"))
	if desc := strings.TrimSpace(f.TaskDescription); desc != "" {
		fmt.Fprintf(&b, "- **所属任务**：%s\n", desc)
	}
	fmt.Fprintf(&b, "- **发现时间**：%s\n", f.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- **生成时间**：%s\n\n", generatedAt.Format("2006-01-02 15:04:05"))
	if s := strings.TrimSpace(f.Summary); s != "" {
		fmt.Fprintf(&b, "## 概述\n\n%s\n\n", s)
	}
	if e := strings.TrimSpace(f.Evidence); e != "" {
		fmt.Fprintf(&b, "## 证据\n\n```\n%s\n```\n\n", e)
	}
	if rep := strings.TrimSpace(f.Report); rep != "" {
		b.WriteString("## 详细报告\n\n")
		b.WriteString(rep)
		b.WriteString("\n")
	}
	b.WriteString(findingTrafficMarkdown(f, true))
	return b.String()
}

var unsafeFilenameChars = regexp.MustCompile(`[^\p{Han}\p{L}\p{N}._-]+`)

// FindingFilename 为「一漏洞一文件」生成安全的 .md 文件名,形如
// `critical_SQL注入_#123.md`。去掉路径分隔符与控制字符,避免 zip 内非法路径。
// 한국어 해설: 분류·제목·ID를 파일명으로 만들고 위험 문자를 치환한다. 마지막 120바이트 절단은 문자 수 제한과 다르다.
func FindingFilename(f *db.DBFinding) string {
	sev := nz(f.Severity, "info")
	title := findingTitle(f)
	name := fmt.Sprintf("%s_%s_#%d", sev, title, f.ID)
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if name == "" {
		name = fmt.Sprintf("finding_%d", f.ID)
	}
	// 防御性:再剥一层路径,杜绝 zip slip。
	name = path.Base(name)
	if len(name) > 120 {
		name = name[:120]
	}
	return name + ".md"
}

// FindingsCSV 把一批 findings 渲染成 CSV(带 UTF-8 BOM,便于 Excel 正确识别中文)。
// 不含大段 report/evidence 全文,只放摘要类字段;需要全文用 Markdown/JSON 导出。
// 한국어 해설: UTF-8 BOM과 CSV writer로 요약 열을 내보낸다. 긴 증거/보고서 전문 대신 바인딩 개수·ID만 넣는다.
func FindingsCSV(fs []*db.DBFinding) []byte {
	items := append([]*db.DBFinding(nil), fs...)
	sortFindingsForExport(items)

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ID", "名称", "类别", "严重等级", "状态", "所属任务", "发现时间", "概述", "流量证据数量", "流量证据ID"})
	for _, f := range items {
		_ = w.Write([]string{
			fmt.Sprintf("%d", f.ID),
			findingTitle(f),
			f.VulnClass,
			nz(f.Severity, "info"),
			nz(f.Status, "pending"),
			f.TaskDescription,
			f.CreatedAt.Format("2006-01-02 15:04:05"),
			strings.TrimSpace(f.Summary),
			fmt.Sprint(len(f.TrafficBindings)), findingTrafficIDs(f),
		})
	}
	w.Flush()
	return buf.Bytes()
}

// 한국어 해설: 연결된 증거 바인딩 ID를 쉼표로 연결한다. 원시 traffic ID와 혼동하지 않도록 한다.
func findingTrafficIDs(f *db.DBFinding) string {
	ids := make([]string, 0, len(f.TrafficBindings))
	for _, b := range f.TrafficBindings {
		ids = append(ids, fmt.Sprint(b.ID))
	}
	return strings.Join(ids, ",")
}

// 한국어 해설: 증거 버전 불일치 경고와 각 바인딩의 역할·URL·상태를 렌더링한다. attachments=true이면 상대 첨부 링크도 추가한다.
func findingTrafficMarkdown(f *db.DBFinding, attachments bool) string {
	stale := f.Report != "" && f.EvidenceVersion != f.ReportEvidenceVersion
	if len(f.TrafficBindings) == 0 && !stale {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n## 关联流量证据\n\n")
	fmt.Fprintf(&out, "证据版本：%d；绑定数量：%d。\n\n", f.EvidenceVersion, len(f.TrafficBindings))
	if stale {
		out.WriteString("证据已变更，详细报告待更新。\n\n")
	}
	for i, b := range f.TrafficBindings {
		fmt.Fprintf(&out, "%d. **证据 #%d · %s** — `%s %s`，状态码 %d\n", i+1, b.ID, b.Role, b.Snapshot.Method, strings.ReplaceAll(b.Snapshot.URL, "`", "%60"), b.Snapshot.Status)
		if b.Note != "" {
			fmt.Fprintf(&out, "   %s\n", strings.ReplaceAll(b.Note, "\n", "\n   "))
		}
		if attachments {
			fmt.Fprintf(&out, "   [请求报文](evidence/%d/%d/request.http) · [响应报文](evidence/%d/%d/response.http)\n", f.ID, b.ID, f.ID, b.ID)
		}
	}
	out.WriteString("\n")
	return out.String()
}
