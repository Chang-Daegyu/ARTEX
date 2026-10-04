# 실행·빌드·설정·저장소 관리

[문서 안내](README.md) · [원본 출처](../../UPSTREAM.md)

이 문서는 한국어 해설판의 소스를 읽고 로컬 환경에서 빌드하는 방법을 설명합니다.
명령은 실행 안내이며, 이번 문서 작성 과정에서 실제 대상 점검이나 앱 운영을 수행했다는
뜻은 아닙니다. 수행한 검증은 [검증 기록](validation.md)에 있습니다.

## 1. 소스와 배포본의 차이

| 경로 | 실행되는 코드 | 이 저장소와의 관계 |
| --- | --- | --- |
| `go build ./cmd/artex` | 현재 체크아웃의 Go 소스 | 한국어판 소스 기준 |
| `npm run build:static` 후 `-tags embedui` | 현재 웹 소스와 Go 소스 | 정적 UI 포함 바이너리 |
| 기본 `docker compose up` | `autumn27/artex` 이미지 | 원저자 배포본 |
| 원본 Releases ZIP | 원저자가 만든 플랫폼 바이너리 | 한국어판을 새로 빌드하지 않음 |
| 페이지 자동 업데이트 | `selfupdate`에 지정한 원본 릴리스 | 자체 빌드를 덮어쓸 수 있음 |

Go의 모듈 경로가 `github.com/Autumn-27/artex`여도 로컬 체크아웃의 해당 모듈 소스를
빌드합니다. GitHub 소유자 경로와 Go 모듈 경로가 반드시 같아야 실행되는 것은 아닙니다.
모듈 경로를 바꾸려면 프로젝트 전체 import와 linker 경로·문서를 일괄 확인해야 하므로
이번 문서화 변경에서는 그대로 보존합니다.

## 2. 필요한 구성 요소

- Go: [go.mod](../../go.mod)에 지정한 `1.26.3` 기준.
- Node.js/npm: 프런트 빌드용. 원본 CI의 Node.js 22를 재현 기준으로 삼을 수 있습니다.
- PostgreSQL: 원본 Compose는 `postgres:16-alpine`입니다.
- LLM: 환경변수 또는 UI에서 공급자와 API 키를 설정합니다.
- 선택 도구: 실행할 Skill과 MCP가 필요로 하는 Python, Chromium, 셸 도구 등.

백엔드가 단일 바이너리라고 해서 PostgreSQL·LLM·외부 실행 도구까지 모두 바이너리 안에
포함되는 것은 아닙니다. `CGO_ENABLED=0`도 도구 실행을 없애는 옵션이 아닙니다.

## 3. 설정 파일 해석

[config/config.go](../../config/config.go)는 DB와 Skill 디렉터리를 읽습니다.
`config.example.json`의 `_comment*` 필드는 사람용 설명입니다. 이 한국어판에서는
그 필드의 설명만 번역했고 실제 연결값과 예제 구조를 보존했습니다.

| 설정 | 의미·우선순위 |
| --- | --- |
| `ARTEX_CONFIG` | 다른 설정 파일 경로 지정 |
| `ARTEX_PG_DSN` | DB 연결 설정 최우선 |
| `database.dsn` | 파일에서 개별 DB 필드보다 우선하는 완전한 DSN |
| `database.host/port/user/password/dbname/sslmode` | DSN이 없을 때 개별 연결 설정 |
| `ARTEX_SKILL_DIR` | 파일의 `skill_dir`보다 우선 |
| `skill_dir` | 비어 있으면 기본 위치 사용, 상대 경로는 작업 디렉터리 기준 |
| `ARTEX_LLM_PROVIDER/MODEL/BASE_URL/PROXY` | 환경에서 지정하는 모델 접속 정보 |
| `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` | 공급자 키, UI 설정도 가능 |
| `NEXT_PUBLIC_MOCK` | 프런트 예시 데이터 사용 여부 |
| `NEXT_EXPORT` | 정적 내보내기 빌드 선택 |
| `NEXT_PUBLIC_SSE_BASE` | 다른 출처의 SSE가 필요한 경우 빌드 때 지정 |

설정 예제의 PostgreSQL 포트는 `5433`이며 Compose 내부 기본은 `5432`입니다.
그대로 복사한 값이 자신의 실행 환경과 맞는지 확인하세요. 예제의 `/opt/artex/skills`도
설치 위치 예시입니다. 빈 문자열로 바꾸면 기본 디렉터리를 사용합니다.

## 4. 스크립트 전체 안내

| 파일 | 진입과 주요 동작 | 알아둘 경계 |
| --- | --- | --- |
| [install.sh](../../install.sh) | Docker 설치 확인 → 배포 방식 선택 → DB 설정 → 시작 | 기본 이미지는 원본, 일부 경로가 시스템 도구 설치 수행 |
| [build.sh](../../build.sh) | 옵션 → 프런트 정적 빌드 → Go 대상별 빌드 → ZIP·체크섬 | 일반 모드/릴리스 모드 공유, `rsync` 필요 |
| [dev.sh](../../dev.sh) | Go와 Next 개발 서버를 자식으로 실행 → 함께 종료 | 개발용이며 앱 격리 수단이 아님 |
| [start.sh](../../start.sh) | Unix 자식 실행 → 신호 전달 → 종료 코드별 재시작 | 다운로드와 파일 교체는 Go 패키지 담당 |
| [start.bat](../../start.bat) | Windows에서 같은 재시작 계약 | 기존 UTF-8 코드 페이지/CRLF 규칙 유지 |
| [update.sh](../../update.sh) | 선택적 Git 갱신 → Docker 교체/로컬 빌드 | 로컬 빌드 뒤 프로세스 재시작 필요 |
| [reset-password.sh](../../reset-password.sh) | DB 연결 → 새 암호 확인 → pgcrypto bcrypt → settings 갱신 | 실제 DB 수정 도구이며 독립적인 관리자 작업 |

`install.sh`의 로컬 분기는 바이너리를 직접 실행합니다. 페이지 업데이트 후의 자동 재실행을
원한다면 시작 스크립트 계약을 이해하고 `start.sh`/`start.bat`로 운영해야 합니다.
원본 설치·업데이트 스크립트의 `cp` 경로는 상위 `server/webui` 준비 여부에 영향을 받을 수
있으므로 이 README의 수동 빌드는 `mkdir -p`를 명시합니다.

## 5. 소스 빌드와 개발

정적 UI 포함 빌드는 루트 README에 있습니다. 개발 모드는 프런트와 백엔드를 분리합니다.

```bash
# web/package-lock.json대로 설치합니다.
cd web
npm ci
cd ..

./dev.sh
```

기본 접속은 `http://localhost:5173`이고 API는 `8787`로 전달됩니다.
개별 실행하려면 `go run ./cmd/artex`와 `cd web && npm run dev`를 각각 사용합니다.
Mock 화면은 DB·LLM 없이 구조를 탐색하는 데 유용하지만 실제 백엔드 파서나 증거 해시
동작을 그대로 재현하는 시험 환경은 아닙니다.

### 릴리스 빌드 옵션

| 옵션/변수 | 효과 |
| --- | --- |
| `--release` | 기본 다중 플랫폼 ZIP 생성 |
| `--target linux/amd64` | 특정 대상 선택 |
| `ARTEX_TARGETS` | 쉼표로 구분한 대상 목록 |
| `ARTEX_BUILD_VERSION` | 바이너리·ZIP 이름에 넣을 버전 |
| `ARTEX_OUTPUT` | 단일 대상 바이너리 출력 경로 |
| `ARTEX_OUTPUT_DIR`, `ARTEX_PACKAGE_DIR` | 바이너리와 ZIP의 디렉터리 |
| `ARTEX_SKIP_FRONTEND=1` | 이미 만든 embed 정적 파일 사용 |
| `ARTEX_SKIP_NPM_CI=1` | 기존 설치 의존성을 재사용 |
| `--upx`, `ARTEX_COMPRESS` | 선택적 UPX 압축 |

자세한 기본값과 상호작용은 [build.sh](../../build.sh)의 한국어 함수 설명과 원문 usage를
함께 보세요. `ARTEX_SKIP_FRONTEND=1`인데 정적 파일이 없으면 실패하는 것이 정상입니다.

### 자체 Docker 이미지

Dockerfile이 요구하는 경로에 현재 소스의 바이너리를 준비한 예시입니다.

```bash
# 앞서 프런트 정적 파일을 server/webui/dist에 준비한 상태입니다.
mkdir -p dist/amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embedui -o dist/amd64/artex ./cmd/artex
docker build --platform linux/amd64 --build-arg TARGETARCH=amd64 -t artex-korean:local .
```

이 명령은 이미지를 만드는 단계입니다. 실행할 때 기본 Compose의 이미지 선택을
`artex-korean:local`로 바꾸는 별도 구성과 DB·데이터 마운트 설정이 필요합니다.
원본의 공개 이미지 이름으로 자신의 이미지를 게시하는 절차는 아닙니다.

## 6. 저장소 자동화·설정 파일

| 파일 | 해석 |
| --- | --- |
| [.github/workflows/release.yml](../../.github/workflows/release.yml) | `v*` 태그 push에 릴리스 빌드·업로드. 일반 PR 테스트 CI와 다름 |
| [Dockerfile](../../Dockerfile) | 사전 빌드 바이너리와 실행 도구 설치. Node/Python은 실행 도구용 |
| [docker-compose.yml](../../docker-compose.yml) | DB와 앱의 기본 배포 |
| [docker-compose.bench.yml](../../docker-compose.bench.yml) | 원본 미추적 `bench/`, `Dockerfile.bench`를 요구하는 벤치마크 참고 구성 |
| [.gitattributes](../../.gitattributes) | 배치 파일 CRLF, 셸 파일 LF 규칙 |
| [.gitignore](../../.gitignore) | 비밀·로컬 설정·산출물 제외. 한국어판은 `docs/` 제외를 해제 |
| [.dockerignore](../../.dockerignore) | Docker 빌드 컨텍스트에서 개발 산출물·민감 로컬 파일 제외 |
| [go.sum](../../go.sum) | Go 의존성 무결성 체크섬. 설명을 넣는 일반 문서가 아님 |
| [web/package-lock.json](../../web/package-lock.json) | npm 의존성의 해석 결과·무결성. 임의 주석을 넣지 않음 |

원본 릴리스 워크플로의 Docker Hub 이미지명과 Secrets는 독립 저장소에 자동 복제되는
계정 설정이 아닙니다. 자체 릴리스를 발행하려면 본인의 배포 대상과 자격 증명을 별도로
설정해야 합니다. 이번 이관에서는 코드를 복사했으며 태그 발행·이미지 배포는 하지 않습니다.

## 7. 검증의 수준

주석 수정의 핵심 검증은 “동작 코드가 같은가”입니다. Go parser/scanner로 문법과
비주석 토큰을 비교하고, JS/TS는 AST를 비교하면 문장 사이에 주석을 넣다가 문자열이나
연산자를 바꾼 실수를 확인할 수 있습니다. 이 검증은 프로그램 성능·외부 서비스 연결의
보증과는 다릅니다.

실제 변경 기능을 검증할 때는 관련 테스트만 실행하되, DB 테스트에는 새 전용 DB를 씁니다.
원본 테스트는 DB가 없으면 일부가 skip되므로 결과의 skip 여부도 확인해야 합니다.

```bash
# 전용 DB를 준비한 뒤 해당 테스트 프로세스에만 DSN을 제공하는 예시입니다.
# 기존 운영 DB의 DSN으로 이 테스트를 실행하지 마세요.
ARTEX_PG_DSN='postgres://test_user:test_password@127.0.0.1:5432/artex_test?sslmode=disable' go test ./db -count=1

# 프런트 의존성을 설치한 뒤 타입·정적 내보내기를 검증할 수 있습니다.
cd web
npx tsc --noEmit
npm run build:static
```

## 8. 원본 변경을 나중에 가져오기

이 저장소는 원본 전체 Git 이력을 이어받은 Fork가 아니라 독립 소스 스냅샷입니다.
원본의 최신 코드를 무조건 덮어쓰면 한국어 설명이 사라지거나 코드와 맞지 않게 됩니다.

1. [UPSTREAM.md](../../UPSTREAM.md)의 기준 커밋을 출발점으로 삼습니다.
2. 원본의 이후 diff를 별도 작업 사본에서 검토합니다.
3. 필요한 기능 변경과 관련 테스트를 함께 적용합니다.
4. 영향을 받은 함수·파일의 한국어 주석과 문서를 갱신합니다.
5. 실제 검증 결과를 남기고 기준 커밋 또는 반영 내역을 기록합니다.

원본 이력이 없는 독립 main에 원본 main을 무조건 merge하는 명령을 기본 업데이트
절차로 사용하지 않습니다. 모듈 경로·자동 업데이트·이미지 레지스트리는 소스 갱신과
서로 다른 관리 항목입니다.
