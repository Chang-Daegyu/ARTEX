// [한국어 파일 안내] notify/mask_test.go
// 비밀의 API 표시와 부분 설정 갱신의 유지·삭제·대상 변경 규칙을 검증한다.
// 대상만 바꾸어 저장된 자격 증명을 새 서버로 보내는 변경은 거절하고 정상 편집·타입 차이·반복 저장은 통과시킨다.
// 테스트 문자열은 모의 자격 증명이며 실제 외부 전송 없이 map과 오류만 확인한다.
package notify

import (
	"errors"
	"strings"
	"testing"
)

// 한국어 해설: 긴 비밀은 주소/전체 토큰을 감추고 마지막 6바이트 힌트와 센티널만 남기는지 확인한다.
func TestMaskedValueHidesBodyButKeepsTailHint(t *testing.T) {
	const secret = "https://oapi.dingtalk.com/robot/send?access_token=abcdef123456"
	got := MaskedValue(secret)
	if strings.Contains(got, "abcdef123456") {
		t.Fatalf("掩码值泄露了完整凭据: %q", got)
	}
	if strings.Contains(got, "oapi.dingtalk.com") {
		t.Fatalf("掩码值不应暴露地址主体: %q", got)
	}
	// 末 6 位要保留，用户才能认出是哪个机器人。
	if !strings.HasSuffix(got, "123456") {
		t.Fatalf("应保留末 6 位作为辨识提示: %q", got)
	}
	if !IsMasked(got) {
		t.Fatalf("掩码值必须能被 IsMasked 识别: %q", got)
	}
}

// 한국어 해설: 6바이트 이하 비밀에 마지막 힌트를 주어 전체가 노출되는 경우를 방지하는지 검사한다.
func TestMaskedValueShortSecretGivesNoHint(t *testing.T) {
	// 短凭据如果也暴露末 6 位，等于把整个凭据暴露出去。
	for _, s := range []string{"abc", "abcdef", ""} {
		got := MaskedValue(s)
		if got != MaskedPrefix {
			t.Fatalf("长度 %d 的凭据不应给出尾部提示，得到 %q", len(s), got)
		}
		if s != "" && strings.Contains(got, s) {
			t.Fatalf("掩码值包含了原值: %q", got)
		}
	}
}

// 한국어 해설: 채널이 선언한 비밀 필드만 마스킹하고 일반 설정의 표시 값은 보존하는지 확인한다.
func TestMaskConfigMasksOnlySecrets(t *testing.T) {
	cfg := map[string]any{
		"webhook": "https://example.com/hook?token=SECRETVALUE",
		"secret":  "SECtest123456",
		"port":    float64(587),
		"host":    "smtp.example.com",
	}
	masked := MaskConfig(KindDingTalk, cfg)
	for _, k := range []string{"webhook", "secret"} {
		s, _ := masked[k].(string)
		if !IsMasked(s) {
			t.Errorf("%s 应被掩码，得到 %q", k, s)
		}
	}
	// 非凭据字段必须原样保留，否则 UI 无法展示。
	if masked["port"] != float64(587) {
		t.Errorf("非凭据字段 port 不应改动: %v", masked["port"])
	}
}

// 한국어 해설: 알 수 없는 채널에서 비밀 필드를 추측하지 않고 빈 설정을 반환하는지 확인한다.
func TestMaskConfigUnknownKindReturnsEmpty(t *testing.T) {
	// 渠道类型无法识别时，宁可让 UI 显示空配置，也不要把可能含凭据的原始内容吐回去。
	got := MaskConfig("nope", map[string]any{"webhook": "https://x/y?token=LEAK"})
	if len(got) != 0 {
		t.Fatalf("未知渠道类型应返回空配置，得到 %v", got)
	}
}

// 한국어 해설: 표시용 마스킹이 실제 저장값을 들고 있는 입력 map을 변경하지 않는지 검사한다.
func TestMaskConfigDoesNotMutateInput(t *testing.T) {
	// 掩码是展示层行为，不能反过来把库里的真值改掉。
	cfg := map[string]any{"webhook": "https://example.com/hook", "secret": "SECtest123456"}
	_ = MaskConfig(KindDingTalk, cfg)
	if IsMasked(cfg["secret"].(string)) {
		t.Fatal("MaskConfig 修改了入参，会导致真实凭据被掩码值覆盖")
	}
}

// 한국어 해설: 마스킹된 비밀을 그대로 제출하면 원래 비밀을 유지하고 다른 변경 필드만 적용하는지 확인한다.
func TestMergeConfigKeepsStoredOnMaskedIncoming(t *testing.T) {
	stored := map[string]any{"webhook": "https://real/hook", "secret": "REALSECRET", "method": "POST"}
	// 用户只改了 method，浏览器提交的是掩码值+新 method。
	incoming := map[string]any{
		"webhook": MaskedValue("https://real/hook"),
		"secret":  MaskedValue("REALSECRET"),
		"method":  "PUT",
	}
	got := MergeConfig(stored, incoming)
	if got["webhook"] != "https://real/hook" || got["secret"] != "REALSECRET" {
		t.Fatalf("掩码字段应保留库中原值，得到 %v", got)
	}
	if got["method"] != "PUT" {
		t.Fatalf("被修改的字段应生效，得到 %v", got["method"])
	}
}

// 한국어 해설: 빈 문자열은 해당 키 삭제이고 미제공 필드는 유지라는 부분 갱신 규칙을 검증한다.
func TestMergeConfigEmptyStringClears(t *testing.T) {
	stored := map[string]any{"webhook": "https://real/hook", "secret": "REALSECRET"}
	got := MergeConfig(stored, map[string]any{"secret": ""})
	if _, ok := got["secret"]; ok {
		t.Fatalf("空串应清空该字段，得到 %v", got)
	}
	// 没提到的字段保留（局部更新语义）。
	if got["webhook"] != "https://real/hook" {
		t.Fatalf("未提及的字段应保留，得到 %v", got)
	}
}

// 한국어 해설: 수정에 언급하지 않은 호스트/암호는 그대로 남고 제공한 포트만 바뀌는지 확인한다.
func TestMergeConfigKeepsUnmentionedStoredKeys(t *testing.T) {
	stored := map[string]any{"host": "smtp.example.com", "port": float64(587), "password": "pw"}
	got := MergeConfig(stored, map[string]any{"port": float64(465)})
	if got["host"] != "smtp.example.com" || got["password"] != "pw" {
		t.Fatalf("未提及的字段应保留，得到 %v", got)
	}
	if got["port"] != float64(465) {
		t.Fatalf("已提及的字段应更新，得到 %v", got["port"])
	}
}

// TestPrepareConfigUpdateBlocksDestinationSwap 是本包最重要的一条安全不变量：
// **改目标地址不能把旧凭据带过去**。
//
// 这些用例用的正是攻击形状的输入（只改地址、对凭据避而不谈），
// 而不是「防御逻辑的正确输入」——只测后者的话，防御没生效也照样全绿。
// 한국어 해설: Webhook·Telegram·SMTP·서명 채널에서 대상만 바꾸거나 이전 마스크를 재사용하면 누락 비밀 키를 명시해 거절하는지 확인한다.
func TestPrepareConfigUpdateBlocksDestinationSwap(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		stored   map[string]any
		incoming map[string]any
		// wantMissing 是预期被点名的凭据键。
		wantMissing string
	}{
		{
			name: "通用 Webhook 改地址想沿用 Authorization 头",
			kind: KindWebhook,
			stored: map[string]any{
				"url":     "https://legit.example.com/hook",
				"headers": map[string]any{"Authorization": "Bearer REAL-TOKEN"},
			},
			incoming:    map[string]any{"url": "https://attacker.tld/c"},
			wantMissing: "headers",
		},
		{
			name:        "Telegram 改 base_url 想把 Bot Token 发到自己的端点",
			kind:        KindTelegram,
			stored:      map[string]any{"bot_token": "123456:REAL", "chat_id": "1", "base_url": "https://api.telegram.org"},
			incoming:    map[string]any{"base_url": "https://attacker.tld"},
			wantMissing: "bot_token",
		},
		{
			name:        "邮件改 SMTP 主机想交出密码",
			kind:        KindEmail,
			stored:      map[string]any{"host": "smtp.corp.com", "port": 587, "password": "REALPW", "from": "a@b.c", "to": []any{"d@e.f"}},
			incoming:    map[string]any{"host": "smtp.attacker.tld"},
			wantMissing: "password",
		},
		{
			name:        "邮件关掉 TLS 也必须重新表态密码",
			kind:        KindEmail,
			stored:      map[string]any{"host": "smtp.corp.com", "port": 587, "tls": false, "password": "REALPW", "from": "a@b.c", "to": []any{"d@e.f"}},
			incoming:    map[string]any{"tls": true},
			wantMissing: "password",
		},
		{
			// 掩码值 = 「沿用旧凭据」，在地址变更的语境下同样必须拒绝。
			name:        "回传掩码凭据 + 新地址",
			kind:        KindTelegram,
			stored:      map[string]any{"bot_token": "123456:REAL", "chat_id": "1", "base_url": "https://api.telegram.org"},
			incoming:    map[string]any{"base_url": "https://attacker.tld", "bot_token": MaskedValue("123456:REAL")},
			wantMissing: "bot_token",
		},
		{
			name:        "钉钉改 Webhook 想沿用加签密钥",
			kind:        KindDingTalk,
			stored:      map[string]any{"webhook": "https://oapi.dingtalk.com/robot/send?access_token=OLD", "secret": "REALSEC"},
			incoming:    map[string]any{"webhook": "https://attacker.tld/hook"},
			wantMissing: "secret",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := PrepareConfigUpdate(tc.kind, tc.stored, tc.incoming)
			if err == nil {
				t.Fatalf("改地址却未重新表态凭据，应当被拒绝；得到配置 %v", merged)
			}
			var target *ErrDestinationChangedWithoutCredentials
			if !errors.As(err, &target) {
				t.Fatalf("应返回专门的错误类型以便接口给出可操作提示，得到 %T: %v", err, err)
			}
			found := false
			for _, m := range target.Missing {
				if m == tc.wantMissing {
					found = true
				}
			}
			if !found {
				t.Fatalf("应点名缺失的凭据键 %q，得到 %v", tc.wantMissing, target.Missing)
			}
			// 错误信息要能指导操作者怎么修。
			if !strings.Contains(err.Error(), tc.wantMissing) {
				t.Errorf("错误信息应提到 %q: %v", tc.wantMissing, err)
			}
		})
	}
}

// TestPrepareConfigUpdateAllowsLegitimateEdits 反向用例：正常的编辑不能被误拦，
// 否则这个防护会因为「太烦」而被绕过或删掉。
// 한국어 해설: 목적지 미변경·새 비밀 명시·비밀 명시 삭제·일반 수신자 변경 같은 정상 편집이 불필요하게 거절되지 않는지 확인한다.
func TestPrepareConfigUpdateAllowsLegitimateEdits(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		stored   map[string]any
		incoming map[string]any
	}{
		{
			name:     "只改名字（配置原样回传）",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://legit.example.com/hook", "headers": map[string]any{"Authorization": "Bearer REAL"}},
			incoming: map[string]any{"url": MaskedValue("https://legit.example.com/hook")},
		},
		{
			name:     "只改请求方法，地址与凭据都不动",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://legit.example.com/hook", "method": "POST"},
			incoming: map[string]any{"method": "PUT"},
		},
		{
			name:     "换地址并**同时**给新凭据",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://old.example.com/hook", "headers": map[string]any{"Authorization": "Bearer OLD"}},
			incoming: map[string]any{"url": "https://new.example.com/hook", "headers": map[string]any{"Authorization": "Bearer NEW"}},
		},
		{
			name:     "换地址并显式声明不再需要凭据",
			kind:     KindWebhook,
			stored:   map[string]any{"url": "https://old.example.com/hook", "headers": map[string]any{"Authorization": "Bearer OLD"}},
			incoming: map[string]any{"url": "https://new.example.com/hook", "headers": ""},
		},
		{
			name:     "Telegram 改 chat_id（不是目的地）",
			kind:     KindTelegram,
			stored:   map[string]any{"bot_token": "t", "chat_id": "1", "base_url": "https://api.telegram.org"},
			incoming: map[string]any{"chat_id": "-100200"},
		},
		{
			name:     "邮件改收件人（不是目的地）",
			kind:     KindEmail,
			stored:   map[string]any{"host": "smtp.corp.com", "port": 587, "password": "PW", "from": "a@b.c", "to": []any{"x@y.z"}},
			incoming: map[string]any{"to": []any{"new@y.z"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := PrepareConfigUpdate(tc.kind, tc.stored, tc.incoming)
			if err != nil {
				t.Fatalf("合法编辑被误拦: %v", err)
			}
			if merged == nil {
				t.Fatal("应返回合并结果")
			}
		})
	}
}

// TestPrepareConfigUpdatePortTypeTolerance 覆盖一个容易误判的细节：
// 前端提交的端口是 JSON number（float64），库里读回来也是 float64，
// 但两个值的类型可能不同（如 int vs float64）。用 == 比较会把「没改」判成「改了」，
// 从而对只改了名字的用户弹出「请重新填写密码」——假警报会让人不再信任这个防护。
// 한국어 해설: 값이 같은 int/float64 포트는 동일하고 실제 다른 포트는 대상 변경으로 판단하는지 검사한다.
func TestPrepareConfigUpdatePortTypeTolerance(t *testing.T) {
	stored := map[string]any{"host": "smtp.corp.com", "port": float64(587), "password": "PW"}
	// 同一个端口，以 int 形式提交。
	if _, err := PrepareConfigUpdate(KindEmail, stored, map[string]any{"port": 587}); err != nil {
		t.Fatalf("端口值相同（仅类型不同）不应被判为地址变更: %v", err)
	}
	// 真的换了端口则必须拦。
	if _, err := PrepareConfigUpdate(KindEmail, stored, map[string]any{"port": 25}); err == nil {
		t.Fatal("端口变更应被拦下")
	}
}

// TestPrepareConfigUpdateSurvivesRepeatedSaveWithBlankDestination 覆盖「可留空的
// 目的地字段」这条路径：Telegram 的 base_url 留空表示用官方 API 地址。
//
// 曾经这里会让渠道从第二次保存起永久保存失败：
//
//	新建时库里存下 base_url:""（创建路径直接存前端提交的 config，不走 MergeConfig）
//	→ 第一次保存，MergeConfig 把空串当显式清空、delete 掉该键
//	→ 第二次保存，incoming 仍是 ""、而 stored 里已经没这个键，被判成「地址变了」
//	→ bot_token 是掩码回显值 → 400「目标地址已变更，请同时重新填写凭据字段」
//
// 用户什么都没改，却从此再也存不上，除非重新粘贴一遍 Bot Token。
// 한국어 해설: 빈 base_url 키가 첫 저장에 삭제되어도 다음 동일 제출이 가짜 대상 변경으로 실패하지 않는지 확인한다.
func TestPrepareConfigUpdateSurvivesRepeatedSaveWithBlankDestination(t *testing.T) {
	stored := map[string]any{"bot_token": "123:ABC", "chat_id": "-100", "base_url": ""}

	// 前端 buildConfig() 对该渠道的每个字段定义都提交一个值：凭据回填掩码，
	// 空文本框提交空串。这里完整复现它的输出，而不是只提交「改动的键」。
	submit := func() map[string]any {
		return map[string]any{
			"bot_token": MaskedValue("123:ABC"),
			"chat_id":   "-100",
			"base_url":  "",
		}
	}

	// 第一次保存：只改了渠道名字，config 原样回传。
	merged, err := PrepareConfigUpdate(KindTelegram, stored, submit())
	if err != nil {
		t.Fatalf("第一次保存被误拦: %v", err)
	}
	if _, ok := merged["base_url"]; ok {
		t.Fatal("前提已变：空串应被 MergeConfig 删除——本用例要覆盖的正是「键消失之后」那一步")
	}

	// 第二次保存：提交内容与上次完全一致，用户什么都没改。
	merged2, err := PrepareConfigUpdate(KindTelegram, merged, submit())
	if err != nil {
		t.Fatalf("第二次保存被误拦（用户什么都没改）: %v", err)
	}
	// 第三次，确认不是「只错一次」而是稳定可保存。
	if _, err := PrepareConfigUpdate(KindTelegram, merged2, submit()); err != nil {
		t.Fatalf("第三次保存被误拦: %v", err)
	}
	// 凭据必须一路保留下来，没有被空串逻辑连带清掉。
	if got := merged2["bot_token"]; got != "123:ABC" {
		t.Fatalf("Bot Token 应沿用原值，得到 %v", got)
	}
}

// TestPrepareConfigUpdateStillGuardsBlankDestinationChanges 是上一条用例的配对
// 断言：把空串与「键不存在」视为等价，**不能**连带放过真正的地址变更。
// 这两个方向都是真实的凭据外发路径——Telegram 的 Bot Token 走在 URL 路径里，
// 换了 base_url 就等于把 Token 送给新地址。
// 한국어 해설: 빈 주소/키 없음의 동등 처리가 실제 기본 API↔사용자 API 변경까지 허용하지 않는지 검증한다.
func TestPrepareConfigUpdateStillGuardsBlankDestinationChanges(t *testing.T) {
	// 方向一：从「空」（官方地址）换到自建地址。
	official := map[string]any{"bot_token": "123:ABC", "chat_id": "-100"}
	if _, err := PrepareConfigUpdate(KindTelegram, official, map[string]any{
		"bot_token": MaskedValue("123:ABC"),
		"base_url":  "https://tg-proxy.attacker.tld",
	}); err == nil {
		t.Fatal("从官方地址换到自建地址必须要求重新填写 Token")
	}

	// 方向二：把自建地址清空（= 换回官方 API）同样是地址变更。
	proxied := map[string]any{"bot_token": "123:ABC", "base_url": "https://proxy.internal/bot"}
	if _, err := PrepareConfigUpdate(KindTelegram, proxied, map[string]any{
		"bot_token": MaskedValue("123:ABC"),
		"base_url":  "",
	}); err == nil {
		t.Fatal("清空自建地址（换回官方 API）同样是地址变更，必须要求重新填写 Token")
	}
}

// 한국어 해설: 모든 채널에 자격 증명의 도착 위치와 비밀 필드가 선언되어 보호가 빠지지 않는지 확인한다.
func TestDestinationKeysDeclaredForEveryKind(t *testing.T) {
	// 与 SecretKeys 同理：渠道若忘记声明目的地键，PrepareConfigUpdate 就保护不到它。
	for kind, ch := range registry {
		if len(ch.DestinationKeys()) == 0 {
			t.Errorf("渠道 %s 未声明目的地键，改地址带出凭据的防护对它无效", kind)
		}
		if len(ch.SecretKeys()) == 0 {
			t.Errorf("渠道 %s 未声明凭据键", kind)
		}
	}
}

// 한국어 해설: 지원 채널마다 적어도 하나의 비밀 키가 등록되어 브라우저에 그대로 노출되지 않는지 확인한다.
func TestSecretKeysDeclaredForEveryKind(t *testing.T) {
	// 编译器已经强制每个渠道实现 SecretKeys，这里再确认一遍「没有渠道在掩码上
	// 交白卷」——返回空切片的渠道意味着它的凭据会明文回显到浏览器。
	expect := map[string]bool{
		KindDingTalk: true, KindFeishu: true, KindWeCom: true,
		KindWebhook: true, KindTelegram: true, KindEmail: true,
	}
	for kind, ch := range registry {
		if !expect[kind] {
			t.Errorf("渠道 %s 未在测试中登记掩码预期", kind)
			continue
		}
		if len(ch.SecretKeys()) == 0 {
			t.Errorf("渠道 %s 未声明任何凭据字段，其配置会明文回显", kind)
		}
	}
}

// TestPrepareConfigUpdateRejectsMaskedInContainer 覆盖审计指出的一处口子：
// 把掩码哨兵塞进**非字符串**结构（如 webhook.headers 是个对象）时，
// MergeConfig 只认「字符串且带前缀」为掩码，于是字面量 "__masked__" 会被当成
// 真实头值存进库——后续鉴权静默失效，且没有任何报错。
// 한국어 해설: headers 객체 안에 마스크 센티널을 실제 값으로 저장하려는 입력은 거절하고 실제 새 객체는 받는지 검증한다.
func TestPrepareConfigUpdateRejectsMaskedInContainer(t *testing.T) {
	stored := map[string]any{
		"url":     "https://legit.example.com/hook",
		"headers": map[string]any{"Authorization": "Bearer REAL"},
	}
	// 对象内部夹带掩码哨兵。
	incoming := map[string]any{
		"headers": map[string]any{"Authorization": MaskedPrefix},
	}
	if _, err := PrepareConfigUpdate(KindWebhook, stored, incoming); err == nil {
		t.Fatal("结构体内部夹带掩码哨兵应被拒绝（否则会把字面量存进库）")
	}
	// 整体提交对象（真实新值）照常接受。
	ok := map[string]any{"headers": map[string]any{"Authorization": "Bearer NEW"}}
	if _, err := PrepareConfigUpdate(KindWebhook, stored, ok); err != nil {
		t.Fatalf("正常提交新请求头不应被拦: %v", err)
	}
}
