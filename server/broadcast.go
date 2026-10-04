// [한국어 길잡이] 작업별 실시간 활동 배포
// 하나의 작업 ID에 여러 SSE 구독 채널을 연결하고 mutex로 등록·해제를 보호한다.
// 구독당 버퍼는 256개이며 Publish는 채널이 가득 차면 이벤트를 버린다. 느린 브라우저가 엔진을 막지 않게 하는 선택이다.
// 활동의 영속 원본은 PostgreSQL이다. server.go의 streamActivity가 연결 시 DB를 재생하지만 연결 중 누락을 모두 자동 복구한다고 보장할 수는 없다.
// 원문에 저장소와 스트림이 항상 같다는 표현이 있으나, 실제 구현은 비차단·손실 가능 실시간 전달이다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"sync"

	"github.com/Autumn-27/artex/db"
)

// Broadcaster is a per-task in-process pub/sub for live activity events. The
// engine publishes each appended activity at its single emit point; SSE handlers
// subscribe per task. Storage (activity table) and the live stream come from the
// same Publish call, so they never diverge.
type Broadcaster struct {
	mu   sync.Mutex
	subs map[string]map[chan db.Activity]struct{} // task id -> set of subscriber channels
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{subs: map[string]map[chan db.Activity]struct{}{}}
}

// Subscribe returns a buffered channel of activities for a task plus an
// unsubscribe func the caller must invoke (defer) to release it.
// [한국어 함수 설명] 작업별 256개 버퍼 채널을 등록한다. 반환된 해제 함수는 sync.Once로 한 번만 채널을 제거하고 닫아 중복 close를 피한다.
func (b *Broadcaster) Subscribe(task string) (<-chan db.Activity, func()) {
	ch := make(chan db.Activity, 256)
	b.mu.Lock()
	if b.subs[task] == nil {
		b.subs[task] = map[chan db.Activity]struct{}{}
	}
	b.subs[task][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			b.mu.Lock()
			if m := b.subs[task]; m != nil {
				delete(m, ch)
				if len(m) == 0 {
					delete(b.subs, task)
				}
			}
			b.mu.Unlock()
			close(ch)
		})
	}
}

// Publish fans an activity out to all subscribers of a task. Non-blocking: if a
// subscriber's buffer is full the event is dropped — the client reconnects with
// its last seq cursor and catches up the gap from the DB, so liveness never
// stalls the engine.
// [한국어 함수 설명] 모든 구독자에게 비차단 전송을 시도한다. default 분기는 버퍼가 찬 구독자의 이벤트를 버리므로 영속 이벤트 큐로 사용할 수 없다.
func (b *Broadcaster) Publish(task string, a db.Activity) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[task] {
		select {
		case ch <- a:
		default:
		}
	}
}
