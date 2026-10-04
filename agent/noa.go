package agent

// 한국어 파일 해설: agent/noa.go
// 선택 기능인 Norma noa 컨텍스트 압축 어댑터를 세션 옵션에 연결한다.
// 설정 콜백은 매 run 시작 때 읽히며 nil 또는 false이면 기본 압축 경로를 그대로 둔다.
// 아카이브는 공통 workDir/noa/<SessionID> 아래에 모아 intent별 작업 폴더와 구분한다.
// Enable 실패는 작업을 중단하지 않고 경고 후 기본 압축으로 돌아간다.
// Enable 성공 시 기본 Compaction을 비워 두 압축기가 동시에 설정되는 상황을 피한다.
// 이 어댑터 호출은 archive/session/warn을 넘기며 실제 모델 창 크기는 여기서 명시적으로 전달하지 않는다.

import (
	"log"
	"path/filepath"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/noaadapter"
)

// noaWarn returns a diagnostics sink tagging non-fatal noa messages with the
// session, routed through the package logger (agents have no per-instance one).
func noaWarn(session string) func(string) {
	return func(msg string) { log.Printf("[noa] %s: %s", session, msg) }
}

// noa 是 norma v0.4.0 引入的「模型驱动上下文压缩」机制,作为平台实验功能由用户在
// 系统设置中开关。它与内置 compaction 互斥:noaadapter.Enable 是唯一入口,一次挂上
// 上下文接管器(Compactor)、Compress 工具与三段常驻提示词,不调用 Enable 即为关闭
// (内置 compaction 照常工作)。开关由每个 agent 注入的 noaEnabledFn 解析,每 run 读
// 一次,故切换只影响之后启动的 run,无需重建 agent。

// enableNoa 在解析器报告开启时把 noa 接入 opts。archiveRoot 是压缩原文的持久化基目录
// (取全局 workDir,各 agent 统一落在 <workDir>/noa 下,不随任务/意图目录分散),sessionID
// 命名其下的归档子目录(全局唯一,故同一基目录内不冲突)。
//
// noa 是实验功能:接入失败不得中断真实任务。发生错误时经 onWarn 上报并回退内置压缩。
// 启用成功时清掉 opts.Compaction,避免 agentcore 因「两个上下文管理器同时设置」告警。
// 한국어: 설정이 켜졌을 때만 어댑터를 적용하고 실패하면 기본 압축을 유지한다.
// 한국어: 고정 SDK 버전의 noa 기본값을 확인해야 하며 이 코드가 실제 창 크기를 자동 전달한다고 가정하지 않는다.
func enableNoa(opts *agentcore.Options, enabled func() bool, archiveRoot, sessionID string, onWarn func(string)) {
	if enabled == nil || !enabled() {
		return
	}
	if opts.OnWarn == nil {
		opts.OnWarn = onWarn
	}
	if err := noaadapter.Enable(opts, noaadapter.Options{
		ArchiveBaseDir: filepath.Join(archiveRoot, "noa"),
		SessionID:      sessionID,
		OnWarn:         onWarn,
	}); err != nil {
		if onWarn != nil {
			onWarn("noa 压缩启用失败,回退内置压缩:" + err.Error())
		}
		return
	}
	// Compactor 覆盖 Compaction,但两者并存时 agentcore 每次会告警;明确清掉。
	opts.Compaction = nil
}
