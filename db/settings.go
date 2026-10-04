package db

// 한국어 읽기 안내
// settings 테이블의 문자열 key/value를 읽고 쓰는 공통 저장 API다. 기능별 세부 설정은 각 호출자가 직렬화해 저장한다.
// GetSetting은 존재 여부를 bool로 따로 반환하며, GetBool은 누락·해석 실패 때 전달받은 기본값을 사용한다.
// SetSetting의 upsert는 같은 키를 갱신한다. 파일 기반 부트스트랩 설정과 DB 런타임 설정의 적용 시점은 config/server 쪽을 함께 확인한다.

import "database/sql"

// Settings is a tiny key-value store for global app config the UI toggles at
// runtime (e.g. traffic_capture). Missing keys fall back to caller defaults.

// GetSetting returns the stored value and ok=false when the key is unset.
func (d *DB) GetSetting(key string) (value string, ok bool, err error) {
	err = d.QueryRow(`SELECT value FROM settings WHERE key=$1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// SetSetting upserts a setting value.
func (d *DB) SetSetting(key, value string) error {
	_, err := d.Exec(`
INSERT INTO settings(key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, key, value)
	return err
}

// GetBool returns the boolean setting, or def when unset/unparseable.
func (d *DB) GetBool(key string, def bool) bool {
	v, ok, err := d.GetSetting(key)
	if err != nil || !ok {
		return def
	}
	return v == "true" || v == "1"
}

// SetBool stores a boolean setting as "true"/"false".
func (d *DB) SetBool(key string, val bool) error {
	if val {
		return d.SetSetting(key, "true")
	}
	return d.SetSetting(key, "false")
}
