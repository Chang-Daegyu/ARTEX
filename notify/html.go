// [한국어 파일 안내] notify/html.go
// 메일용 HTML 본문을 간단한 인라인 스타일로 만든다.
// 본문의 텍스트와 href 속성은 서로 다른 이스케이프 규칙을 사용한다.
// 태그·이미지를 신뢰되지 않은 발견 제목으로 주입하지 못하도록 &, <, >를 치환하고 속성에서는 따옴표도 처리한다.
package notify

import (
	"fmt"
	"strings"
)

// 本文件渲染邮件的 HTML 正文。刻意用内联样式 + 简单表格布局而不是现代 CSS：
// 邮件客户端（尤其 Outlook 与国内企业邮箱）对 <style> 块和 flex/grid 的支持
// 差异极大，内联样式是唯一在各家都能正确显示的写法。

// htmlSeverityColor 返回级别对应的强调色，用于左侧色条与标题。
// 한국어 해설: 심각도에 맞는 강조색을 반환한다. 알 수 없는 값은 중립 회색을 사용한다.
func htmlSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "#d32029"
	case "high":
		return "#e8830c"
	case "medium":
		return "#d4b106"
	case "low":
		return "#1677ff"
	default:
		return "#8c8c8c"
	}
}

// htmlTitle 返回邮件主题。
// 한국어 해설: 형식 중립적인 공통 제목을 메일 제목으로 재사용한다.
func htmlTitle(m Message) string {
	return markdownTitle(m)
}

// htmlBody 渲染邮件正文 HTML。maxRunes<=0 表示不截断。
// 한국어 해설: 단일/묶음 항목과 플랫폼 링크를 HTML로 합친다. maxRunes=0인 메일 경로는 전체 본문을 유지한다.
func htmlBody(m Message, maxRunes int) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Microsoft YaHei',sans-serif;font-size:14px;color:#262626;line-height:1.6;">`)
	if m.Batch {
		b.WriteString(htmlBatchIntro(m))
		for _, it := range m.Items {
			b.WriteString(htmlItem(it, false))
		}
	} else if len(m.Items) > 0 {
		b.WriteString(htmlItem(m.Items[0], true))
	}
	if m.HomeURL != "" {
		fmt.Fprintf(&b, `<p style="margin:16px 0 0;"><a href="%s" style="color:#1677ff;">在平台中查看全部</a></p>`, htmlEscapeAttr(m.HomeURL))
	}
	b.WriteString(`</div>`)
	return TruncateHTML(b.String(), maxRunes)
}

// htmlBatchIntro 渲染汇总邮件开头：条数与级别分布。
// 한국어 해설: 시간창과 전체 항목 수 및 심각도 분포를 메일 머리말로 렌더링한다.
func htmlBatchIntro(m Message) string {
	var b strings.Builder
	if m.WindowMinutes > 0 {
		fmt.Fprintf(&b, `<h2 style="font-size:16px;margin:0 0 4px;">近 %d 分钟新增 %d 个漏洞</h2>`, m.WindowMinutes, len(m.Items))
	} else {
		fmt.Fprintf(&b, `<h2 style="font-size:16px;margin:0 0 4px;">新增 %d 个漏洞</h2>`, len(m.Items))
	}
	counts := map[string]int{}
	for _, it := range m.Items {
		counts[it.Severity]++
	}
	var parts []string
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		if n := counts[sev]; n > 0 {
			parts = append(parts, fmt.Sprintf(`<span style="color:%s;font-weight:600;">%s %d</span>`,
				htmlSeverityColor(sev), htmlEscape(SeverityLabel(sev)), n))
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(&b, `<p style="margin:0 0 12px;">%s</p>`, strings.Join(parts, " &middot; "))
	}
	return b.String()
}

// htmlItem 渲染单个漏洞。full=true 时含摘要与回链（单条推送），
// false 时压缩成一行（汇总列表）。
// 한국어 해설: 단일 메시지의 상세 블록 또는 묶음의 짧은 블록을 만들고 텍스트/속성 위치별로 이스케이프한다.
func htmlItem(it Item, full bool) string {
	color := htmlSeverityColor(it.Severity)
	var b strings.Builder
	if full {
		fmt.Fprintf(&b, `<div style="border-left:4px solid %s;padding:8px 0 8px 12px;margin-bottom:12px;">`, color)
	} else {
		fmt.Fprintf(&b, `<div style="border-left:3px solid %s;padding:4px 0 4px 10px;margin-bottom:8px;">`, color)
	}
	fmt.Fprintf(&b, `<div style="font-weight:600;">%s &middot; %s</div>`,
		htmlEscape(SeverityLabel(it.Severity)), htmlEscape(it.Title()))

	if !full {
		var extras []string
		if a := assetLine(it.Assets, maxAssetsShown); a != "" {
			extras = append(extras, htmlEscape(a))
		}
		if it.Summary != "" {
			extras = append(extras, htmlEscape(OneLine(it.Summary, 60)))
		}
		if len(extras) > 0 {
			fmt.Fprintf(&b, `<div style="color:#595959;font-size:13px;">%s</div>`, strings.Join(extras, " &middot; "))
		}
		b.WriteString(`</div>`)
		return b.String()
	}

	if it.IsStatusChange() {
		fmt.Fprintf(&b, `<div><b>状态变更</b>：%s → %s</div>`,
			htmlEscape(StatusLabel(it.FromStatus)), htmlEscape(StatusLabel(it.ToStatus)))
	}
	if it.VulnClass != "" && it.VulnClass != it.Title() {
		fmt.Fprintf(&b, `<div><b>类型</b>：%s</div>`, htmlEscape(it.VulnClass))
	}
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		fmt.Fprintf(&b, `<div><b>资产</b>：%s</div>`, htmlEscape(a))
	}
	if s := OneLine(it.Summary, maxSummaryRunes); s != "" {
		fmt.Fprintf(&b, `<div><b>摘要</b>：%s</div>`, htmlEscape(s))
	}
	if it.DetailURL != "" {
		fmt.Fprintf(&b, `<div style="margin-top:6px;"><a href="%s" style="color:#1677ff;">查看详情</a></div>`, htmlEscapeAttr(it.DetailURL))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// htmlEscape 转义 HTML 文本内容。漏洞标题与摘要来自被测目标与模型输出，
// 是不可信内容——不转义就等于允许把任意 HTML（含外链图片）注入到邮件里。
// 한국어 해설: HTML 텍스트 위치에서 구조를 만들 수 있는 &, <, >를 순서대로 엔티티로 바꾼다.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// htmlEscapeAttr 转义 HTML 属性值（在文本转义之外额外处理引号，
// 防止 URL 里的引号提前闭合 href 属性）。
// 한국어 해설: 텍스트 이스케이프에 큰따옴표 처리까지 더하여 URL이 href 속성 경계를 닫지 못하게 한다.
func htmlEscapeAttr(s string) string {
	s = htmlEscape(s)
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
