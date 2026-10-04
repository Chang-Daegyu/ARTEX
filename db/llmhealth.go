package db

// 한국어 읽기 안내
// LLM 프로필의 일시적 장애와 냉각 종료 시각을 저장해 프로세스 재시작 뒤에도 회로 차단 상태를 이어간다.
// LoadLLMHealth는 아직 만료되지 않은 냉각만 반환한다. 이미 시간이 지난 장애를 재시작과 함께 다시 활성화하지 않는다.
// Save는 프로필 단위 upsert이고 Clear는 수동 복구 요청 시 해당 상태를 삭제한다. 실제 요청 선택은 llmpool 계층에서 수행한다.

import "time"

// LLM failover health: one row per profile tracking its circuit-breaker state.
// The authoritative copy lives in memory (llmpool.Registry); this table only
// survives a restart so a cooling-off window isn't silently reset by one.

// LLMHealth is one profile's circuit-breaker state.
type LLMHealth struct {
	ProfileID int64      `json:"profile_id"`
	Fails     int        `json:"fails"`      // consecutive failures; cleared on success
	Trips     int        `json:"trips"`      // total trips, drives the backoff ladder
	OpenUntil *time.Time `json:"open_until"` // nil/past = closed (healthy)
	LastError string     `json:"last_error"`
	LastAt    time.Time  `json:"last_at"`
}

// LoadLLMHealth returns the profiles still in an UNEXPIRED cooling-off window.
// Expired rows are deliberately skipped: after a restart a profile that has
// finished cooling should be treated as healthy again and re-probed on its next
// call, not resurrected as broken.
// 한국어: 현재 시각보다 냉각 종료가 늦은 행만 복구한다. 만료된 장애를 재시작 때 다시 실패 상태로 되살리지 않는다.
func (d *DB) LoadLLMHealth() ([]LLMHealth, error) {
	rows, err := d.Query(`SELECT profile_id,fails,trips,open_until,COALESCE(last_error,''),last_at
FROM llm_profile_health WHERE open_until IS NOT NULL AND open_until > now()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LLMHealth
	for rows.Next() {
		var h LLMHealth
		if err := rows.Scan(&h.ProfileID, &h.Fails, &h.Trips, &h.OpenUntil, &h.LastError, &h.LastAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// SaveLLMHealth upserts one profile's circuit-breaker state.
func (d *DB) SaveLLMHealth(h LLMHealth) error {
	_, err := d.Exec(`
INSERT INTO llm_profile_health(profile_id,fails,trips,open_until,last_error,last_at)
VALUES ($1,$2,$3,$4,$5,now())
ON CONFLICT (profile_id) DO UPDATE SET
  fails=EXCLUDED.fails, trips=EXCLUDED.trips, open_until=EXCLUDED.open_until,
  last_error=EXCLUDED.last_error, last_at=now()`,
		h.ProfileID, h.Fails, h.Trips, h.OpenUntil, h.LastError)
	return err
}

// ClearLLMHealth drops one profile's state (manual "recover now" from the UI).
func (d *DB) ClearLLMHealth(profileID int64) error {
	_, err := d.Exec(`DELETE FROM llm_profile_health WHERE profile_id=$1`, profileID)
	return err
}
