// Package config loads runtime configuration from a JSON file, with environment
// variables taking precedence. Currently it carries the PostgreSQL connection.
// [한국어 길잡이] 설정 경로와 PostgreSQL 연결 문자열
// 설정 파일 위치는 ARTEX_CONFIG → 현재 디렉터리의 config.json → 실행 파일 옆 config.json 순서로 찾는다.
// 연결 문자열은 ARTEX_PG_DSN이 최우선이며, 다음으로 database.dsn 또는 database의 개별 필드로 조립한다.
// BaseDir은 배포 바이너리 위치를 기준으로 하되 go run의 임시/캐시 실행 파일은 현재 작업 디렉터리를 기준으로 한다.
// Load는 읽기·JSON 오류를 상위로 반환하지 않고 기본값을 사용한다. 실제 DB 설정이 없으면 PostgresDSN이 명시적인 오류를 반환한다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Database is the PostgreSQL connection config. Either set DSN directly, or set
// the component fields and a DSN is assembled from them.
type Database struct {
	DSN      string `json:"dsn"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// Config is the on-disk config file shape.
type Config struct {
	Database Database `json:"database"`
	SkillDir string   `json:"skill_dir"`
}

// BaseDir is the directory that anchors all runtime artifacts (config.json and
// the data/ store). It is the directory the running binary lives in, so a
// distributed executable keeps its files next to itself on any OS (Windows,
// Linux, …) regardless of the working directory it is launched from.
//
// When launched via `go run`, the binary is throwaway: it sits either in a temp
// build dir (cache miss → fresh link) OR straight inside the Go build cache
// (cache hit → run from $GOCACHE/.../...-d). We detect both and fall back to the
// current working directory so dev artifacts (data/, transcripts) and config
// resolve against the project dir, not the throwaway binary's location.
// [한국어 함수 설명] 실행 파일 옆을 배포의 기준 경로로 사용한다. go run 위치로 판정되면 임시 빌드 디렉터리에 데이터를 남기지 않도록 현재 디렉터리를 선택한다.
func BaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	if isGoRunDir(dir) {
		return "." // throwaway `go run` binary → use CWD
	}
	return dir
}

// isGoRunDir reports whether dir is where `go run` parked its executable: under
// the system temp dir (cache miss), or anywhere inside a Go build cache
// (…/go-build/…, the cache-hit case — NOT under os.TempDir(), which is why the
// old temp-only check failed intermittently). In both cases the binary is
// throwaway, so config/data must resolve against the CWD.
// [한국어 함수 설명] 시스템 임시 디렉터리 아래이거나 go-build 경로 구성 요소를 포함하는지 검사한다. 실제 실행 방식 그 자체를 조회하는 것이 아니라 경로에 따른 휴리스틱이다.
func isGoRunDir(dir string) bool {
	if tmp := os.TempDir(); tmp != "" {
		if rel, err := filepath.Rel(tmp, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	for _, seg := range strings.Split(filepath.ToSlash(dir), "/") {
		if seg == "go-build" {
			return true
		}
	}
	return false
}

// Path returns the config file path. Resolution order:
//  1. env ARTEX_CONFIG (explicit override)
//  2. ./config.json in the current working directory (running from the project
//     dir — robust no matter where `go run` placed the temp/cached binary)
//  3. config.json next to the executable (a distributed binary keeps it beside)
//
// The first existing file wins. If none exist, the CWD path is returned so the
// "not found" message points at the project dir the user most likely expected.
// [한국어 함수 설명] 명시적 환경 변수는 존재 여부와 관계없이 반환한다. 기본 후보 중에는 실제 존재하는 첫 파일을 선택하고, 없으면 예상 위치를 오류 안내에 쓴다.
func Path() string {
	if v := strings.TrimSpace(os.Getenv("ARTEX_CONFIG")); v != "" {
		return v
	}
	var candidates []string
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "config.json"))
	}
	candidates = append(candidates, filepath.Join(BaseDir(), "config.json"))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}

// Load reads and parses the config file. A missing/unreadable file yields a zero
// Config (so callers fall back to defaults) rather than an error.
// [한국어 함수 설명] 설정 읽기와 JSON 디코딩 실패를 오류로 반환하지 않는 느슨한 로더다. 필수 DB 설정 여부는 이후 PostgresDSN에서 결정한다.
func Load() Config {
	var c Config
	b, err := os.ReadFile(Path())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

// SkillDir returns the skill root directory with precedence:
//
//	env ARTEX_SKILL_DIR  >  config file (skill_dir)  >  BaseDir()/skills
//
// The directory is created if it does not exist.
// [한국어 함수 설명] 환경 변수·JSON 설정·기본 경로 중 하나를 고르고 디렉터리 생성을 시도한다. MkdirAll 오류는 이 함수에서 반환하지 않는다.
func SkillDir() string {
	var d string
	if v := strings.TrimSpace(os.Getenv("ARTEX_SKILL_DIR")); v != "" {
		d = v
	} else if v := strings.TrimSpace(Load().SkillDir); v != "" {
		d = v
	} else {
		d = filepath.Join(BaseDir(), "skills")
	}
	_ = os.MkdirAll(d, 0o755)
	return d
}

// PostgresDSN resolves the connection string with precedence:
//
//	env ARTEX_PG_DSN  >  config file (database.dsn, or assembled from fields)
//
// There is NO built-in fallback: when neither source supplies a database config,
// it returns an error naming the config path it inspected, so startup fails loudly
// instead of silently connecting to a wrong default. source describes where the
// DSN came from (for startup logging).
// [한국어 함수 설명] 우선순위대로 DSN과 출처 설명을 함께 반환한다. 연결 정보를 전혀 찾지 못하면 임의의 DB에 접속하지 않도록 오류를 낸다.
func PostgresDSN() (dsn, source string, err error) {
	if v := strings.TrimSpace(os.Getenv("ARTEX_PG_DSN")); v != "" {
		return v, "环境变量 ARTEX_PG_DSN", nil
	}
	db := Load().Database
	if d := strings.TrimSpace(db.DSN); d != "" {
		return d, "配置文件 " + Path() + " (database.dsn)", nil
	}
	if db.Host != "" || db.DBName != "" || db.User != "" {
		return db.buildDSN(), "配置文件 " + Path() + " (database 字段)", nil
	}
	return "", "", fmt.Errorf("未找到数据库配置：环境变量 ARTEX_PG_DSN 未设置，且配置文件 %s 未提供 database（dsn 或 host/user/dbname）。请创建该配置文件或设置环境变量后重试", Path())
}

// [한국어 함수 설명] 누락된 host/port/sslmode의 기본값을 채운 뒤 net/url로 사용자명·비밀번호와 쿼리를 인코딩한다. 이 함수는 DB 접속을 시도하지 않는다.
func (d Database) buildDSN() string {
	host := d.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := d.Port
	if port == 0 {
		port = 5432
	}
	ssl := d.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		Host:   host + ":" + strconv.Itoa(port),
		Path:   "/" + d.DBName,
	}
	if d.User != "" {
		if d.Password != "" {
			u.User = url.UserPassword(d.User, d.Password)
		} else {
			u.User = url.User(d.User)
		}
	}
	u.RawQuery = url.Values{"sslmode": {ssl}}.Encode()
	return u.String()
}

// String is a redacted view of the resolved DSN (password masked) for logging.
// [한국어 함수 설명] URL 형식 DSN의 비밀번호를 별표로 바꾼 로그용 표현을 만든다. URL 파싱 실패 시 원문을 반환하므로 임의 형식의 비밀 제거기로 일반화하면 안 된다.
func Redact(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		if _, hasPw := u.User.Password(); hasPw {
			u.User = url.UserPassword(u.User.Username(), "****")
		}
	}
	return fmt.Sprintf("%s", u.String())
}
