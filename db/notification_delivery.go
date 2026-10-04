package db

// 한국어 읽기 안내
// 알림 전달 작업의 선점·임대·재시도·완료·이력 조회를 관리한다. 네트워크 전송 자체는 notify 패키지의 dispatcher가 담당한다.
// 이 큐는 FOR UPDATE SKIP LOCKED와 sending 임대를 사용한다. exploration.go의 worker intent 선점은 별도의 조건부 UPDATE 방식이다.
// 채널별 허용량에 맞춰 선점한 뒤 attempts를 올린다. 요약 메시지에 아직 실리지 못한 항목은 DeferDeliveries가 횟수를 되돌려 실패 예산을 낭비하지 않는다.
// 임대가 만료된 sending은 재선점 가능하다. 외부 채널 전송과 DB 확정 사이에는 중복 전달 가능성을 고려해야 한다.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// 本文件是投递任务的领取与状态流转。
//
// 领取用「租约」而非长事务：把行置为 sending 并把 next_attempt_at 推到未来作为
// 租约到期时间，提交事务后再去做网络投递。这样投递期间不持有数据库锁——
// 网络请求可能耗时数秒（客户端超时 15 秒），占着行锁不放会拖垮同库的其它写操作。
//
// 代价是进程若在投递途中崩溃，行会停在 sending。这是**可自愈**的：租约到期后
// next_attempt_at 落入过去，下一轮领取会把同一行重新捞起来（见领取条件里的
// state IN ('pending','sending')）。重试计数在领取时就已 +1，所以崩溃不会造成
// 无限重试——MaxNotifyAttempts 次机会用完后落入 failed 等人工处理。

// MaxNotifyAttempts 是一条投递的最大尝试次数（含首次）。
// 定义在这里而非投递引擎里：它是状态机自身的策略，引擎只是执行者。
const MaxNotifyAttempts = 3

// MaxDigestBatchSize 是单个汇总批次一次最多合并多少条投递。
//
// 存在的理由是资源：一个汇总周期内如果扫出几万个漏洞（完全可能——一次全量扫描
// 就能做到），不设上界的话领取会把全部行读进内存、渲染成一条超长消息，
// 然后被渠道的长度上限截掉大半——既浪费内存，又**静默丢失**被截掉的那些漏洞。
// 设上界后，超出的部分留在库里成为下一个批次，下个周期自然发出去，不会丢。
//
// 取 500 的依据：它是渲染成消息后在企微 4096 字节上限内还"有内容可读"的量级；
// 再大也只是让截断发生在更靠后的位置而已。
const MaxDigestBatchSize = 500

// NotificationDelivery 是一条投递任务，含渲染所需的渠道配置与事件快照。
type NotificationDelivery struct {
	ID            int64           `json:"id"`
	EventID       int64           `json:"event_id"`
	ChannelID     int64           `json:"channel_id"`
	State         string          `json:"state"`
	Attempts      int             `json:"attempts"`
	NextAttemptAt time.Time       `json:"next_attempt_at"`
	LastError     string          `json:"last_error"`
	BatchID       *int64          `json:"batch_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	SentAt        *time.Time      `json:"sent_at,omitempty"`
	Snapshot      json.RawMessage `json:"snapshot,omitempty"`
	// 联合加载的渲染上下文，不进 JSON（由 server 层组装 DTO）。
	Channel *NotificationChannel `json:"-"`
	// FindingID/EventKind 从事件带出，供历史列表直接跳转漏洞详情。
	FindingID int64  `json:"finding_id,string"`
	EventKind string `json:"event_kind"`
	// ChannelName/ChannelKind 是列表展示用的冗余字段，省掉前端二次查询。
	ChannelName string `json:"channel_name"`
	ChannelKind string `json:"channel_kind"`
}

const notificationDeliveryCols = `d.id, d.event_id, d.channel_id, d.state, d.attempts, d.next_attempt_at,
       d.last_error, d.batch_id, d.created_at, d.sent_at`

// joinedDeliveryQuery 是投递行的统一读取形状：投递 + 事件快照 + 渠道配置。
// 渲染一条消息三者缺一不可，分开查会写出三次往返。
const joinedDeliveryQuery = `SELECT ` + notificationDeliveryCols + `,
       e.snapshot, e.kind, e.finding_id,
       c.id, c.name, c.kind, c.enabled, c.config, c.mode, c.filter, c.rate_per_min
FROM notification_deliveries d
JOIN notification_events e ON e.id = d.event_id
JOIN notification_channels c ON c.id = d.channel_id`

func scanNotificationDelivery(sc interface{ Scan(...any) error }) (*NotificationDelivery, error) {
	var (
		dl        NotificationDelivery
		lastErr   sql.NullString
		batchID   sql.NullInt64
		sentAt    sql.NullTime
		snapshot  []byte
		eventKind string
		channel   NotificationChannel
		chEnabled bool
	)
	if err := sc.Scan(&dl.ID, &dl.EventID, &dl.ChannelID, &dl.State, &dl.Attempts, &dl.NextAttemptAt,
		&lastErr, &batchID, &dl.CreatedAt, &sentAt,
		&snapshot, &eventKind, &dl.FindingID,
		&channel.ID, &channel.Name, &channel.Kind, &chEnabled, &channel.Config, &channel.Mode, &channel.Filter, &channel.RatePerMin); err != nil {
		return nil, err
	}
	dl.LastError = lastErr.String
	if batchID.Valid {
		dl.BatchID = &batchID.Int64
	}
	if sentAt.Valid {
		dl.SentAt = &sentAt.Time
	}
	dl.Snapshot = json.RawMessage(snapshot)
	dl.EventKind = eventKind
	dl.ChannelName = channel.Name
	dl.ChannelKind = channel.Kind
	channel.Enabled = &chEnabled
	dl.Channel = &channel
	return &dl, nil
}

// claimQuery 描述一次领取：先按 sel 选出候选并加锁，再把它们置为 sending 并
// 延长租约。sel 里的 lease 位置由调用方用 $n 占位并自行传参。
type claimQuery struct {
	sql  string
	args []any
}

// ClaimRealtimeDeliveries 领取某渠道一批到期的实时投递，最多 limit 条。
//
// 刻意按**单个渠道**领取而不是「全局领一批再挑着发」：限流闸在投递引擎里按渠道
// 维护，只有先知道这个渠道这一轮还能发几条、再去领同样多的行，限流才不会消耗
// 重试次数。若反过来先领后弃，被限流挡下的行已经被计过一次 attempts，
// 3 次预算会被纯粹的等待耗光，最后落进 failed。
//
// 条件含「租约已过期的 sending」——那是崩溃自愈的落点。lease 必须显著大于单次
// 投递的最坏耗时（渠道 HTTP 客户端超时 15 秒），否则同一行会被两个 dispatcher
// 同时投递。同时挡掉已停用渠道：停用操作已把存量投递标记为 skipped，
// 这里再拦一道，避免停用与领取并发时的漏网。
// 한국어: 채널 한 개의 현재 허용량만큼만 선점한다. 먼저 전역으로 많이 선점한 뒤 버리면 rate limit 대기가 attempts를 소진할 수 있기 때문이다.
func (d *DB) ClaimRealtimeDeliveries(ctx context.Context, channelID int64, limit int, lease time.Duration) ([]*NotificationDelivery, error) {
	if limit <= 0 {
		return nil, nil
	}
	return d.claimDeliveries(ctx, lease, claimQuery{
		sql: `SELECT dd.id FROM notification_deliveries dd
JOIN notification_channels c ON c.id = dd.channel_id
WHERE dd.channel_id = $1 AND dd.state IN ($2,$3) AND dd.next_attempt_at <= now()
  AND c.enabled AND c.mode = $4
ORDER BY dd.next_attempt_at, dd.id
FOR UPDATE OF dd SKIP LOCKED
LIMIT $5`,
		args: []any{channelID, NotifyStatePending, NotifyStateSending, NotifyModeRealtime, limit},
	}, nil)
}

// DigestBatchDue 报告该渠道是否已攒够一个到期批次：存在待发投递，且**最老的那条**
// 年龄已达到汇总周期。
//
// 判定依据是最老投递的年龄而非墙上时钟：这样刚建好的渠道不会因为对齐到整点而
// 立刻吐出一条只有一条的「汇总」，积压很久的批次也不会再白等一轮。
//
// 与 ClaimDigestBatch 分开是因为语义不同：本函数只回答「该不该发」，
// 而领取要拿走该渠道**全部**待发行（包括尚未满年龄的那些）——否则一个周期
// 会被拆成多条消息，汇总就失去意义了。
// 한국어: 가장 오래 기다린 전달 항목의 나이로 요약 발송 시점을 판정한다. 새 채널을 만들었다는 이유만으로 정각에 불완전한 요약을 바로 내보내지 않는다.
func (d *DB) DigestBatchDue(ctx context.Context, channelID int64, minAge time.Duration) (bool, error) {
	var due bool
	err := d.QueryRowContext(ctx, `SELECT EXISTS (
  SELECT 1 FROM notification_deliveries d
  JOIN notification_channels c ON c.id = d.channel_id
  WHERE d.channel_id = $1 AND d.state IN ($2,$3) AND c.enabled
  GROUP BY d.channel_id
  HAVING min(d.created_at) <= now() - make_interval(secs => $4)
)`, channelID, NotifyStatePending, NotifyStateSending, int64(minAge.Seconds())).Scan(&due)
	return due, err
}

// ClaimDigestBatch 领取某渠道当前到期的待发投递，作为一个汇总批次，
// 单批最多 MaxDigestBatchSize 条。
//
// 同批次的所有投递共享 batch_id，用集合里的最小 id 作批次号（稳定、可读、
// 无需额外序列）。重试时用 COALESCE 保留原批次号，使「这批 N 条是一起发的」
// 在多次重试后依然成立。
//
// 按 id 升序取前 N 条而非随机取：最早产生的投递最先发出去，积压时不会出现
// 「新漏洞先发、老漏洞永远排在后面」的饥饿。
// 한국어: 현재 전달 후보를 ID 순으로 한 배치에 묶고 가장 작은 ID를 안정적인 batch_id로 사용한다. 재시도에서도 원래 배치 식별자를 유지한다.
func (d *DB) ClaimDigestBatch(ctx context.Context, channelID int64, limit int, lease time.Duration) ([]*NotificationDelivery, error) {
	if limit <= 0 {
		return nil, nil
	}
	// limit 是**内存上界**，调用方传 MaxDigestBatchSize；这里再夹一道，
	// 防止调用方传进一个更大的值。
	//
	// 刻意不接受「限流额度」充当批次大小：限流的单位是消息条数——一个批次只发
	// 一条消息、消耗一个令牌，由 server 层的 takeTokens 扣除——与「一批装几条
	// 漏洞」是两个不同的量纲。曾经为了让 rate_per_min 对 digest 生效而把每轮
	// 请求预算传进来当批次大小，结果 rate=20/min 的渠道每批只装 1 条漏洞，
	// digest 退化成带汇总文案的实时推送。要改限流请改 takeTokens 的 want，
	// 不要动这里。
	if limit > MaxDigestBatchSize {
		limit = MaxDigestBatchSize
	}
	out, err := d.claimDeliveries(ctx, lease, claimQuery{
		sql: `SELECT dd.id FROM notification_deliveries dd
JOIN notification_channels c ON c.id = dd.channel_id
WHERE dd.channel_id = $1 AND dd.state IN ($2,$3) AND dd.next_attempt_at <= now() AND c.enabled
ORDER BY dd.id
FOR UPDATE OF dd SKIP LOCKED
LIMIT $4`,
		args: []any{channelID, NotifyStatePending, NotifyStateSending, limit},
	}, func(tx *sql.Tx, ids []int64) error {
		batchID := ids[0]
		for _, id := range ids {
			if id < batchID {
				batchID = id
			}
		}
		ph, idArgs := placeholders(2, ids)
		_, err := tx.ExecContext(ctx, `UPDATE notification_deliveries SET batch_id = COALESCE(batch_id, $1)
WHERE id IN (`+ph+`)`, append([]any{batchID}, idArgs...)...)
		return err
	})
	return out, err
}

// claimDeliveries 执行「选取 + 置 sending 延长租约 + 读取完整行」，全在一个事务里。
// postClaim 是可选的附加步骤（汇总批次用它写入 batch_id）。
// 한국어: 선택 행 잠금 → sending 전환/임대 연장/시도 수 증가 → 선택적 배치 보완 → 전체 행 조회를 같은 트랜잭션에서 수행한다.
func (d *DB) claimDeliveries(ctx context.Context, lease time.Duration, cq claimQuery, postClaim func(*sql.Tx, []int64) error) ([]*NotificationDelivery, error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // 提交成功后是 no-op

	ids, err := selectForClaim(ctx, tx, cq.sql, cq.args...)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, tx.Commit()
	}
	// 置 sending 并把 next_attempt_at 推到未来：这个未来时刻即租约到期时间，
	// 「租约未到期」与「未到重试时间」因此共用同一个条件表达，不需要新增列。
	ph, idArgs := placeholders(3, ids)
	if _, err := tx.ExecContext(ctx, `UPDATE notification_deliveries
SET state=$1, attempts=attempts+1, next_attempt_at=now()+make_interval(secs => $2)
WHERE id IN (`+ph+`)`,
		append([]any{NotifyStateSending, lease.Seconds()}, idArgs...)...); err != nil {
		return nil, err
	}
	if postClaim != nil {
		if err := postClaim(tx, ids); err != nil {
			return nil, err
		}
	}
	out, err := loadDeliveriesTx(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit()
}

func selectForClaim(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func loadDeliveriesTx(ctx context.Context, tx *sql.Tx, ids []int64) ([]*NotificationDelivery, error) {
	ph, args := placeholders(1, ids)
	rows, err := tx.QueryContext(ctx, joinedDeliveryQuery+` WHERE d.id IN (`+ph+`) ORDER BY d.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*NotificationDelivery{}
	for rows.Next() {
		dl, err := scanNotificationDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, dl)
	}
	return out, rows.Err()
}

// MarkDeliveriesSent 把一批投递标记为已送达。
func (d *DB) MarkDeliveriesSent(ctx context.Context, ids []int64) error {
	ph, args := placeholders(2, ids)
	if len(args) == 0 {
		return nil
	}
	_, err := d.ExecContext(ctx, `UPDATE notification_deliveries
SET state=$1, sent_at=now(), last_error='' WHERE id IN (`+ph+`)`, append([]any{NotifyStateSent}, args...)...)
	return err
}

// RescheduleDeliveries 把一批投递退回 pending 并推后重试时间。
//
// 退回 pending 而不是引入新的中间状态，是为了让「还剩几次机会」只由一个地方
// 表达（MaxNotifyAttempts），避免状态机的分支随重试策略膨胀。
// 한국어: 실패한 전달을 pending으로 돌리고 다음 시도 시각을 늦춘다. 새로운 중간 상태를 만들기보다 attempts와 due 시각으로 재시도 정책을 표현한다.
func (d *DB) RescheduleDeliveries(ctx context.Context, ids []int64, delay time.Duration, errMsg string) error {
	ph, args := placeholders(4, ids)
	if len(args) == 0 {
		return nil
	}
	_, err := d.ExecContext(ctx, `UPDATE notification_deliveries
SET state=$1, next_attempt_at=now()+make_interval(secs => $2), last_error=$3
WHERE id IN (`+ph+`)`,
		append([]any{NotifyStatePending, delay.Seconds(), truncateNotifyError(errMsg)}, args...)...)
	return err
}

// DeferDeliveries 把一批投递退回 pending、立即可再领，并**撤销领取时计的那一次尝试**。
//
// 用途只有一个：汇总消息按渠道长度上限分段发送时，没装进本条的条目要留到下一批。
// 那不是失败，所以不该消耗重试预算——领取时 attempts 已经乐观地 +1 了，
// 这里必须减回去。否则一个 500 条的积压会按每段 20 条切成 25 段，
// 尾部条目在第 3 段就被 MaxNotifyAttempts 判成 failed，而它们从未出过任何错。
//
// GREATEST(...,0) 兜住「有人手工重发把 attempts 清零后又走到这里」的情况，
// 不让计数变成负数。
// 한국어: 요약 길이 제한으로 이번 메시지에 포함되지 못한 항목을 다시 pending으로 돌리고 attempts를 하나 빼 준다. 아직 발송하지 않은 항목은 실패가 아니다.
func (d *DB) DeferDeliveries(ctx context.Context, ids []int64, reason string) error {
	ph, args := placeholders(3, ids)
	if len(args) == 0 {
		return nil
	}
	_, err := d.ExecContext(ctx, `UPDATE notification_deliveries
SET state=$1, attempts=GREATEST(attempts-1, 0), next_attempt_at=now(), last_error=$2
WHERE id IN (`+ph+`)`,
		append([]any{NotifyStatePending, truncateNotifyError(reason)}, args...)...)
	return err
}

// FailDeliveries 把一批投递标记为最终失败，等待人工在投递历史里重发。
func (d *DB) FailDeliveries(ctx context.Context, ids []int64, errMsg string) error {
	// 占位符从 $3 开始：$1 是 state、$2 是 last_error。
	ph, args := placeholders(3, ids)
	if len(args) == 0 {
		return nil
	}
	_, err := d.ExecContext(ctx, `UPDATE notification_deliveries SET state=$1, last_error=$2 WHERE id IN (`+ph+`)`,
		append([]any{NotifyStateFailed, truncateNotifyError(errMsg)}, args...)...)
	return err
}

// RetryNotificationDelivery 手动重发一条投递：重置为 pending、清零重试计数、
// 立即到期。清计数是刻意的——人工点「重发」意味着前几次失败的原因已被处理，
// 再拿旧计数限制它没有道理。
// 한국어: 사람의 재전송 요청은 시도 횟수를 0으로 돌리고 즉시 다시 대상이 되게 한다. 이전 장애 원인을 처리한 뒤 새 예산으로 시도한다는 의미다.
func (d *DB) RetryNotificationDelivery(ctx context.Context, id int64) error {
	res, err := d.ExecContext(ctx, `UPDATE notification_deliveries
SET state=$2, attempts=0, next_attempt_at=now(), last_error=''
WHERE id=$1 AND state IN ($3,$4)`, id, NotifyStatePending, NotifyStateFailed, NotifyStateSkipped)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("投递 %d 不存在或当前状态不允许重发", id)
	}
	return nil
}

// NotificationDeliveryFilter 是投递历史的查询条件。
type NotificationDeliveryFilter struct {
	ChannelID int64
	State     string
	EventKind string
}

func (f NotificationDeliveryFilter) where() (string, []any) {
	var conds []string
	var args []any
	if f.ChannelID > 0 {
		args = append(args, f.ChannelID)
		conds = append(conds, fmt.Sprintf("d.channel_id=$%d", len(args)))
	}
	if f.State != "" {
		args = append(args, f.State)
		conds = append(conds, fmt.Sprintf("d.state=$%d", len(args)))
	}
	if f.EventKind != "" {
		args = append(args, f.EventKind)
		conds = append(conds, fmt.Sprintf("e.kind=$%d", len(args)))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListNotificationDeliveries 分页返回投递历史，新的在前。
func (d *DB) ListNotificationDeliveries(ctx context.Context, f NotificationDeliveryFilter, page, pageSize int) ([]*NotificationDelivery, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	where, args := f.where()

	var total int
	if err := d.QueryRowContext(ctx, `SELECT count(*) FROM notification_deliveries d
JOIN notification_events e ON e.id = d.event_id`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf("%s%s ORDER BY d.id DESC LIMIT $%d OFFSET $%d",
		joinedDeliveryQuery, where, len(args)+1, len(args)+2)
	rows, err := d.QueryContext(ctx, q, append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*NotificationDelivery{}
	for rows.Next() {
		dl, err := scanNotificationDelivery(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, dl)
	}
	return out, total, rows.Err()
}

// truncateNotifyError 把错误信息截到列可接受的长度。渠道返回的响应体可能很长
// （通用 Webhook 打到自建服务时尤甚），不截断会让历史列表的载荷膨胀。
func truncateNotifyError(msg string) string {
	const max = 500
	if len(msg) <= max {
		return msg
	}
	// 按字符边界回退，避免留下半个 UTF-8 字符让前端显示成乱码。
	cut := max
	for cut > 0 && !isUTF8Start(msg[cut]) {
		cut--
	}
	return msg[:cut] + "…"
}

func isUTF8Start(b byte) bool { return b&0xC0 != 0x80 }

// placeholders 生成从 start 开始的 $n 占位串及对应参数，供 IN (...) 使用。
// 例如 start=3, ids=[7,8] → "$3,$4", [7,8]。
// 한국어: IN 절의 ID를 $start부터 매개변수화한다. 값이 SQL 문장에 직접 결합되지 않으며 인자 순서는 생성한 자리표시자 순서와 일치해야 한다.
func placeholders(start int, ids []int64) (string, []any) {
	ph := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		ph = append(ph, fmt.Sprintf("$%d", start+i))
		args = append(args, id)
	}
	return strings.Join(ph, ","), args
}
