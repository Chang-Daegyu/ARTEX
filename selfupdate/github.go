// [한국어 파일 안내] selfupdate/github.go
// GitHub의 최신 정식 릴리스와 플랫폼별 ZIP 자산을 찾는다.
// 주의: 독립 저장소로 복제해도 Repo 상수는 원본 Autumn-27/artex를 계속 가리킨다.
// 원클릭 업데이트는 원본 릴리스 바이너리를 설치할 수 있으므로 한국어 수정본 배포에는 이 저장소 소스로 직접 빌드하는 절차가 필요하다.
// HTTPS·허용 호스트·체크섬은 전송/파일 무결성 보호이며 원본 릴리스 소유자와 별개인 독립 코드 서명 검증은 아니다.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Repo 是发布源。写死而不是做成配置项：更新源可配等于给任何能改配置的人一条
// 远程代码执行通道，对一个渗透测试平台来说这个口子开不得。
const Repo = "Autumn-27/artex"

// latestURL 是 GitHub 的"最新正式版"接口。它会自动跳过 prerelease 和 draft。
const latestURL = "https://api.github.com/repos/" + Repo + "/releases/latest"

// allowedHosts 限定升级链路能访问的域名。配合下面的 checkRedirect，
// 任何一跳被重定向到名单外的主机都会直接失败——这是防止 DNS 污染 / 中间人
// 把二进制换掉的第一道闸门，第二道是 SHA256SUMS 比对。
var allowedHosts = map[string]bool{
	"api.github.com":                       true,
	"github.com":                           true,
	"objects.githubusercontent.com":        true, // release 资产实际落地的对象存储
	"release-assets.githubusercontent.com": true,
	"raw.githubusercontent.com":            true,
}

// Release 是 GitHub Release 里我们关心的字段。
// 한국어 자료형: GitHub API 응답 중 태그·발행 시각·본문·자산 목록처럼 업데이트에 필요한 정보다.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
}

// Asset 是 Release 上挂的一个文件。
// 한국어 자료형: 첨부 파일의 이름·다운로드 URL·기대 크기다. 본문 바이트는 Stage에서 별도로 내려받는다.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// NewClient 构造一个只认 GitHub 域名的 HTTP 客户端。proxy 为空则直连。
//
// 刻意不复用默认 Transport：升级链路必须强制走 TLS 且校验证书，不能被别处
// 设置的 InsecureSkipVerify 之类影响到。
// 한국어 해설: 다른 코드의 TLS 검증 생략 설정을 물려받지 않는 전용 Transport를 만든다. 리다이렉트마다 HTTPS와 호스트 허용 목록을 재검사한다.
func NewClient(proxy string) *http.Client {
	tr := &http.Transport{
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 15 * time.Second,
	}
	if p := strings.TrimSpace(proxy); p != "" {
		if pu, err := url.Parse(p); err == nil {
			tr.Proxy = http.ProxyURL(pu)
		}
	}
	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Minute, // 下载整包，不能按请求级超时卡死
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			return checkURL(req.URL)
		},
	}
}

// checkURL 强制 https + 域名白名单。
// 한국어 해설: https 스킴과 정확한 허용 호스트 이름을 요구한다. github.com처럼 보이는 접미사 위장 도메인은 허용하지 않는다.
func checkURL(u *url.URL) error {
	if u.Scheme != "https" {
		return fmt.Errorf("拒绝非 HTTPS 地址: %s", u.Scheme+"://"+u.Host)
	}
	if !allowedHosts[strings.ToLower(u.Hostname())] {
		return fmt.Errorf("拒绝非 GitHub 域名: %s", u.Hostname())
	}
	return nil
}

// FetchLatest 查询最新正式版。
// 한국어 해설: 고정된 원본 저장소의 releases/latest를 읽어 태그 존재와 HTTP 상태를 확인한다. 403/429는 제한 오류로 설명한다.
func FetchLatest(ctx context.Context, c *http.Client) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return nil, err
	}
	if err := checkURL(req.URL); err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "artex-selfupdate")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("访问 GitHub 失败（可在系统设置里配置全局代理）: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusTooManyRequests:
		// 未认证的 GitHub API 是每 IP 每小时 60 次，共用出口 IP 时很容易撞上。
		return nil, fmt.Errorf("GitHub 接口限流（每小时 60 次），请稍后再试")
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("仓库 %s 尚未发布任何正式版本", Repo)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("GitHub 返回 %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析 Release 失败: %w", err)
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return nil, fmt.Errorf("Release 缺少 tag")
	}
	return &rel, nil
}

// AssetName 返回当前平台对应的发布包名，与 build.sh 的 package_binary 保持一致：
// artex-<版本>-<os>-<arch>.zip（版本号不带 v 前缀）。
// 한국어 해설: build.sh와 같은 artex-버전-os-arch.zip 이름을 만든다. 태그의 v 접두어는 파일명에서 제외한다.
func AssetName(tag, goos, goarch string) string {
	return fmt.Sprintf("artex-%s-%s-%s.zip", strings.TrimPrefix(tag, "v"), goos, goarch)
}

// FindAsset 在 Release 里按名字找资产。
// 한국어 해설: 릴리스 첨부 목록에서 이름을 대소문자 무시로 비교하여 원하는 자산을 찾는다.
func (r *Release) FindAsset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if strings.EqualFold(a.Name, name) {
			return a, true
		}
	}
	return Asset{}, false
}
