// [한국어 파일 안내] selfupdate/bootstrap.go
// 서버 포트·DB 초기화 전에 준비된 업데이트를 확인하고 파일 교체 또는 복구를 결정한다.
// 준비본이 있으면 검증·실행 확인 후 교체하고, 마커만 있으면 새 버전의 시작 횟수를 누적한다.
// HTTP 시작 뒤 SettleDelay를 살아남아 Settle가 호출되면 마커를 지워 안정화 완료로 간주한다.
// 이는 시작 가능성과 생존 시간의 확인이며 애플리케이션 전체 기능/DB 마이그레이션의 의미적 정상성을 보증하는 검사는 아니다.
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// smokeEnv 让被冒烟测试拉起的子进程直接跳过 Bootstrap。
//
// 严格来说不加也不会出事：子进程的 os.Executable() 是 artex.new，推导出来的
// 全部路径都带 .new 前缀，碰不到真正的升级文件。但依赖这种巧合太脆弱，
// 显式短路一目了然，也省掉子进程一次无谓的磁盘探测。
const smokeEnv = "ARTEX_SELFUPDATE_SMOKE"

// Action 是 Bootstrap 给 main 的指令。
// 한국어 자료형: main이 일반 시작을 계속할지 재실행 종료 코드를 반환할지 결정하는 두 상태다.
type Action int

const (
	// Continue：照常启动 server。
	Continue Action = iota
	// Restart：立刻以 ExitRestart 退出，让守护脚本重新拉起。
	Restart
)

// State 描述本次启动时的升级状态，供 /api/update/check 如实告诉前端
// "上一次升级是成功了还是被回滚了"。
// 한국어 자료형: 미확인 업데이트·자동 복구·준비 실패를 UI에 설명하기 위한 현재 시작 상태다.
type State struct {
	Pending     bool   // 换装后尚未确认稳定
	RolledBack  bool   // 本次启动刚刚执行过自动回滚
	FailedStage bool   // 暂存件校验/冒烟未通过，已丢弃
	Detail      string // 面向用户的一句话说明
}

// Bootstrap 在 main 的最开头运行，必须在任何监听端口、打开数据库之前调用。
//
// 三种局面：
//
//	① 存在暂存件 artex.new  → 校验 + 冒烟，通过则换装并要求重启；不通过则丢弃继续跑旧版
//	② 只剩标记文件          → 说明刚换装完，累计一次尝试；连续失败够多次则回滚
//	③ 什么都没有            → 正常启动
// 한국어 해설: smoke 자식 프로세스는 건너뛰고 .new·마커 유무에 따라 교체/안정화/일반 시작으로 분기한다.
func Bootstrap() (Action, State) {
	if os.Getenv(smokeEnv) != "" {
		return Continue, State{}
	}
	p, err := ResolvePaths()
	if err != nil {
		log.Printf("[update] 跳过自举：%v", err)
		return Continue, State{}
	}

	if _, err := os.Stat(p.New); err == nil {
		return applyStaged(p)
	}

	m, ok := readMarker(p.Marker)
	if !ok {
		return Continue, State{}
	}
	return confirmOrRollback(p, m)
}

// applyStaged 处理"存在暂存件"的局面：校验通过就换装，失败就丢弃。
//
// 这里是整个升级链路唯一会覆盖可执行文件的地方，也是最后一道闸门——冒烟测试挡掉
// 下载损坏、架构选错、动态链接缺失这类问题。一旦放行一个跑不起来的二进制，
// 守护脚本会不知疲倦地反复拉起它，而 Go 代码根本没机会运行，自动回滚也就无从谈起。
// 한국어 해설: 새 파일의 해시와 -h 실행을 확인한 뒤 현재 파일을 백업하고 교체한다. 검증 실패는 준비본을 버리고 현재 버전을 유지한다.
func applyStaged(p Paths) (Action, State) {
	m, _ := readMarker(p.Marker)

	if err := verifyStaged(p); err != nil {
		log.Printf("[update] 暂存的新版本未通过校验，已丢弃，继续运行当前版本：%v", err)
		cleanStaged(p)
		_ = os.Remove(p.Marker)
		return Continue, State{FailedStage: true, Detail: "新版本校验失败，已丢弃：" + err.Error()}
	}

	if err := swap(p); err != nil {
		log.Printf("[update] 换装失败，继续运行当前版本：%v", err)
		cleanStaged(p)
		_ = os.Remove(p.Marker)
		return Continue, State{FailedStage: true, Detail: "换装失败：" + err.Error()}
	}

	// 换装成功。保留标记，交给下一次启动（跑的就是新版）确认是否稳定。
	m.Attempts = 0
	if m.StagedAt == 0 {
		m.StagedAt = time.Now().Unix()
	}
	if err := writeMarker(p.Marker, m); err != nil {
		log.Printf("[update] 写升级标记失败（失去自动回滚能力）：%v", err)
	}
	log.Printf("[update] 已换装到 %s，退出以重启（exit %d）", orUnknown(m.To), ExitRestart)
	return Restart, State{Pending: true}
}

// confirmOrRollback 处理"换装后的启动"：累计尝试次数，超限则把旧版换回来。
//
// 计数只在 Go 代码跑起来后才递增，所以它覆盖的是"能执行但初始化时崩溃"
// （配置不兼容、端口被占、DB 迁移炸了）这类故障；"根本无法 exec" 由换装前的
// 冒烟测试挡住，两者合起来才是完整的。
// 한국어 해설: 새 버전의 시작 횟수를 증가시켜 3회를 넘으면 백업으로 되돌린다. 자동 복구 실패 시 마커를 지워 끝없는 복구 반복을 피한다.
func confirmOrRollback(p Paths, m marker) (Action, State) {
	m.Attempts++
	if m.Attempts > maxAttempts {
		if err := rollback(p); err != nil {
			// 回滚都失败了就别再重启了，否则会陷入无限重启。清掉标记，
			// 让进程按当前状态起——起不来的话用户至少能在日志里看到原因。
			log.Printf("[update] 新版本连续 %d 次启动失败，且回滚失败：%v", maxAttempts, err)
			_ = os.Remove(p.Marker)
			return Continue, State{Detail: "新版本启动失败且回滚失败：" + err.Error()}
		}
		log.Printf("[update] 新版本连续 %d 次启动失败，已回滚到 %s，退出以重启（exit %d）",
			maxAttempts, orUnknown(m.From), ExitRestart)
		_ = os.Remove(p.Marker)
		return Restart, State{RolledBack: true, Detail: fmt.Sprintf("新版本启动失败，已回滚到 %s", orUnknown(m.From))}
	}
	if err := writeMarker(p.Marker, m); err != nil {
		log.Printf("[update] 更新升级标记失败：%v", err)
	}
	log.Printf("[update] 新版本启动中（第 %d/%d 次尝试），稳定运行后将确认升级",
		m.Attempts, maxAttempts)
	return Continue, State{Pending: true}
}

// Settle 确认新版本已稳定运行，清除升级标记。
//
// 由 main 在 HTTP 监听起来之后延迟调用：活过这段时间才算数，否则标记留在原地，
// 下次启动继续累计尝试次数，直到触发回滚。
// 한국어 해설: 현재 실행 파일의 경로를 구한 뒤 안정화 마커 정리를 호출한다. 지연 시간을 기다리는 책임은 main 호출부에 있다.
func Settle() {
	p, err := ResolvePaths()
	if err != nil {
		return
	}
	settle(p)
}

// 한국어 해설: 유효한 업그레이드 마커를 지워 시작 실패 횟수 누적을 끝낸다. .old 백업은 수동 복구용으로 남긴다.
func settle(p Paths) {
	if _, ok := readMarker(p.Marker); !ok {
		return // 不是升级后的启动，无事可做
	}
	if err := os.Remove(p.Marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("[update] 清除升级标记失败：%v", err)
		return
	}
	log.Printf("[update] 新版本运行稳定，升级完成（上一版本保留为 %s）", p.Old)
}

// SettleDelay 是判定"新版本活下来了"所需的运行时长。
const SettleDelay = 30 * time.Second

// verifyStaged 校验暂存件：先比对 SHA256，再真正把它拉起来跑一次。
// 한국어 해설: 준비된 바이너리를 저장된 SHA-256과 비교하고 실제 -h 실행까지 확인한다.
func verifyStaged(p Paths) error {
	want, err := os.ReadFile(p.Sum)
	if err != nil {
		return fmt.Errorf("读取校验和: %w", err)
	}
	got, err := fileSHA256(p.New)
	if err != nil {
		return fmt.Errorf("计算校验和: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(string(want)), got) {
		return errors.New("SHA256 不匹配（下载损坏或被篡改）")
	}
	return smokeTest(p.New)
}

// smokeTest 用 -h 拉起新二进制，确认它在当前系统上真的能执行。
// 这能挡掉下载截断、架构选错（exec format error）、缺依赖等一大类问题。
// 한국어 해설: 실행 권한을 부여하고 30초 제한의 -h 자식 프로세스를 띄운다. 자식은 환경 표식으로 부트스트랩 재진입을 건너뛴다.
func smokeTest(bin string) error {
	if err := os.Chmod(bin, 0o755); err != nil {
		return fmt.Errorf("赋予执行权限: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "-h")
	cmd.Env = append(os.Environ(), smokeEnv+"=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return errors.New("冒烟测试超时（新二进制无响应）")
	}
	if err != nil {
		snippet := strings.TrimSpace(string(out))
		if len(snippet) > 300 {
			snippet = snippet[:300] + "…"
		}
		return fmt.Errorf("冒烟测试失败: %v: %s", err, snippet)
	}
	return nil
}

// swap 把当前二进制换成暂存的新版本。
//
// Unix 和 Windows 都允许 rename 一个正在运行的可执行文件（Windows 禁止的是删除和
// 覆盖，rename 不在其列），所以这里不需要分平台，也不需要先停掉自己。
// 한국어 해설: 이전 백업을 정리하고 현재→.old, .new→현재 순서로 rename한다. 두 번째 이동 실패 시 첫 이동을 복원한다.
func swap(p Paths) error {
	// Windows 的 rename 不会覆盖已存在的目标，上一轮升级留下的 .old 必须先清掉。
	if err := os.Remove(p.Old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("清理旧备份 %s: %w", p.Old, err)
	}
	if err := os.Rename(p.Current, p.Old); err != nil {
		return fmt.Errorf("备份当前版本: %w", err)
	}
	if err := os.Rename(p.New, p.Current); err != nil {
		// 换装失败但当前版本已经被挪走了，必须原样放回去，否则下次启动没有可执行文件。
		if rerr := os.Rename(p.Old, p.Current); rerr != nil {
			return fmt.Errorf("装入新版本失败(%v)，且恢复当前版本失败: %w", err, rerr)
		}
		return fmt.Errorf("装入新版本: %w", err)
	}
	_ = os.Remove(p.Sum)
	return nil
}

// rollback 把 swap 备份的旧版本换回来。
// 한국어 해설: 시작 실패한 새 버전을 .failed로 남기고 .old를 현재 위치로 되돌려 원인 조사 자료를 보존한다.
func rollback(p Paths) error {
	if _, err := os.Stat(p.Old); err != nil {
		return fmt.Errorf("没有可回滚的备份 %s: %w", p.Old, err)
	}
	// 把起不来的新版挪到 .failed 留作排查，而不是直接删掉。
	failed := p.Current + ".failed"
	_ = os.Remove(failed)
	if err := os.Rename(p.Current, failed); err != nil {
		return fmt.Errorf("移走失败的版本: %w", err)
	}
	if err := os.Rename(p.Old, p.Current); err != nil {
		return fmt.Errorf("恢复旧版本: %w", err)
	}
	return nil
}

// Rollback 是 /api/update/rollback 的实现：主动退回上一版本。
// 只做换装，重启同样交给守护脚本（调用方随后以 ExitRestart 退出）。
// 한국어 해설: 사용자가 요청한 수동 복구는 백업을 실행 확인한 후 현재/백업을 서로 교환한다. 재시작 요청은 상위 호출자가 처리한다.
func Rollback() error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	if _, err := os.Stat(p.Old); err != nil {
		return errors.New("没有可回滚的上一版本（" + p.Old + " 不存在）")
	}
	cleanStaged(p)
	if err := smokeTest(p.Old); err != nil {
		return fmt.Errorf("上一版本无法执行，拒绝回滚: %w", err)
	}
	// 交换当前与备份：回滚之后还能再滚回来。
	tmp := p.Current + ".swap"
	_ = os.Remove(tmp)
	if err := os.Rename(p.Current, tmp); err != nil {
		return fmt.Errorf("移走当前版本: %w", err)
	}
	if err := os.Rename(p.Old, p.Current); err != nil {
		_ = os.Rename(tmp, p.Current)
		return fmt.Errorf("装入上一版本: %w", err)
	}
	if err := os.Rename(tmp, p.Old); err != nil {
		log.Printf("[update] 回滚后整理备份失败（不影响运行）：%v", err)
	}
	_ = os.Remove(p.Marker)
	return nil
}

// HasBackup 报告是否存在可回滚的上一版本，供前端决定要不要显示回滚按钮。
// 한국어 해설: 현재 실행 디렉터리에 .old가 존재하는지 확인하여 UI의 복구 버튼 표시 판단에 사용한다.
func HasBackup() bool {
	p, err := ResolvePaths()
	if err != nil {
		return false
	}
	_, err = os.Stat(p.Old)
	return err == nil
}

// 한국어 해설: 파일을 스트리밍하여 전체 내용의 SHA-256을 소문자 16진수로 계산한다.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// 한국어 해설: 비어 있는 버전 문자열만 사용자 표시용 기본 문구로 바꾼다.
func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "未知版本"
	}
	return s
}
