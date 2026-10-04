# 원본 전체 파일 지도

기준 커밋에서 Git이 추적한 **598개 파일 전부**를 경로별로 정리했습니다. 신규 한국어 문서는 [문서 안내](README.md)에서 찾습니다. 원본 파일을 제거하지 않고 설명을 추가했습니다.

Go·TS/TSX·JS/MJS·Python·CSS·SQL·셸·배치와 실행 설정 등 주석 가능한 **541개 파일**에 한국어 설명이 있는지 검사했습니다. JSON·잠금·라이선스·이미지 등은 원문 형식을 유지하고 아래 표와 모듈 문서에서 해설합니다.

파일 경로를 누르면 코드로 이동합니다. 함수별 설명은 소스 본문과 해당 모듈 가이드를 함께 읽으세요. 표의 설명은 빠른 탐색용이며 상세 가이드를 대체하지 않습니다.

## 루트 — 20개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [.dockerignore](<../../.dockerignore>) | Docker 빌드 컨텍스트 제외 규칙. 로컬 데이터·개발 캐시·임시 산출물을 구분합니다. | [가이드](development.md) |
| [.env.example](<../../.env.example>) | Docker 환경변수 예제 실제 사용은 .env로 복사한 뒤 DB 암호를 설정합니다. 원본 기본 이미지 태그와 LLM 공급자 선택값을 지정합니다. | [가이드](development.md) |
| [.gitattributes](<../../.gitattributes>) | Git 텍스트 규칙: *.bat CRLF, *.sh LF. 플랫폼 줄바꿈 계약입니다. | [가이드](development.md) |
| [.gitignore](<../../.gitignore>) | 로컬 설정·비밀·산출물 제외 규칙. 한국어판에서는 docs/ 제외를 해제해 문서를 추적합니다. | [가이드](development.md) |
| [CHANGELOG.md](<../../CHANGELOG.md>) | 원본 모든 버전의 변경 기록을 한국어로 정리합니다. 당시 기록과 현재 구현은 구분합니다. | [가이드](development.md) |
| [Dockerfile](<../../Dockerfile>) | 실행용 컨테이너 구성 CI/로컬에서 미리 빌드한 dist/<TARGETARCH>/artex를 이미지로 복사합니다. 이 Dockerfile 안에서 Go/Next를 컴파일하지 않습니다. | [가이드](development.md) |
| [LICENSE](<../../LICENSE>) | 원본 AGPL-3.0 전문. 번역으로 효력을 대체하지 않으며 README의 한국어 고지와 함께 읽습니다. | [가이드](development.md) |
| [README.md](<../../README.md>) | 한국어판 진입 문서. 설치·구조·용어·문서 안내·라이선스 고지를 설명합니다. | [가이드](development.md) |
| [build.sh](<../../build.sh>) | 릴리스 빌드와 패키징 인자와 환경변수로 대상 OS/아키텍처를 결정하고 프런트엔드를 정적 내보내기한 뒤 Go embed 빌드를 실행합니다. | [가이드](development.md) |
| [config.example.json](<../../config.example.json>) | PostgreSQL·Skill 경로 예제. _comment 설명을 한국어로 번역했으며 실제 설정 값은 유지합니다. | [가이드](development.md) |
| [dev.sh](<../../dev.sh>) | 개발 실행 백엔드와 Next 개발 서버를 같은 프로세스 그룹에서 시작하고 wait로 유지합니다. /api 요청은 프런트에서 백엔드로 전달됩니다. | [가이드](development.md) |
| [docker-compose.bench.yml](<../../docker-compose.bench.yml>) | 원본 벤치마크 구성 참고 기본 Compose와 별도이며 Dockerfile.bench와 bench/.env를 참조하고 host 네트워크를 사용합니다. | [가이드](development.md) |
| [docker-compose.yml](<../../docker-compose.yml>) | 기본 Docker 서비스 구성 postgres 건강 검사가 성공하면 artex를 시작합니다. 앱 DSN은 Compose 서비스명 postgres를 사용하며 DB 포트는 호스트에 별도 공개하지 않습니다. | [가이드](development.md) |
| [go.mod](<../../go.mod>) | Go 모듈 경로·언어 버전·직접/간접 의존성. 원본 경로와 버전을 유지하며 development.md에서 해설합니다. | [가이드](development.md) |
| [go.sum](<../../go.sum>) | Go 의존성 모듈의 체크섬. 버전과 무결성을 고정하는 자료로 원문 그대로 보존합니다. | [가이드](development.md) |
| [install.sh](<../../install.sh>) | 초기 설치 안내 Docker 이미지 실행 또는 로컬 소스 컴파일 중 하나를 선택합니다. DB 연결 정보와 .env/config.json을 준비한 뒤 앱을 시작합니다. | [가이드](development.md) |
| [reset-password.sh](<../../reset-password.sh>) | 관리자 암호 복구 기존 PostgreSQL settings의 auth.password_hash에 bcrypt 해시를 저장합니다. 로컬 psql 또는 Docker 내부 psql 경로를 선택합니다. | [가이드](development.md) |
| [start.bat](<../../start.bat>) | Windows 감시 시작 스크립트 artex.exe를 실행하고 0은 종료, 75는 재시작, 나머지는 최대 60초 대기 후 재시작합니다. | [가이드](development.md) |
| [start.sh](<../../start.sh>) | Unix 감시 시작 스크립트 artex 자식 프로세스를 실행하고 종료 코드 0이면 종료, 75이면 즉시 재시작, 그 외에는 지수적으로 기다린 뒤 재시작합니다. | [가이드](development.md) |
| [update.sh](<../../update.sh>) | 기존 설치 갱신 현재 Git 작업 사본을 선택적으로 fast-forward 갱신한 뒤 Docker 이미지 교체 또는 로컬 재컴파일을 수행합니다. | [가이드](development.md) |

## .github — 1개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [.github/workflows/release.yml](<../../.github/workflows/release.yml>) | 원본 릴리스 자동화 v로 시작하는 태그 push가 프런트 정적 빌드, 5개 플랫폼 바이너리, ZIP/체크섬 릴리스, Docker Hub 업로드를 순서대로 실행합니다. | [가이드](development.md) |

## agent — 52개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [agent/assembly.go](<../../agent/assembly.go>) | 역할별 도구 목록을 실제 실행 가능한 도구 묶음으로 조립한다. 입력은 역할 키(agentKey)와 기본 CoreTool 목록이며, 서버가 주입한 Skill/MCP 도구를 합친다. | [가이드](modules/agents.md) |
| [agent/blackboard_inheritance_test.go](<../../agent/blackboard_inheritance_test.go>) | 직접 연결 작업의 읽기 전용 문맥 상속을 검증한다. 기업 scope 확장, fact/finding/intent 조회와 출처 표시, 연결을 넘는 쓰기 거부, 단계 상세 조회 한도의 안정적인 축소를 검사한다. | [가이드](modules/agents.md) |
| [agent/cancelcause.go](<../../agent/cancelcause.go>) | context 취소에 사람이 이해할 수 있는 원인 코드를 부여하는 공통 사전이다. AbortCause는 기계 판별용 Code, 짧은 표시용 Short, 자세한 경위 Text를 함께 가진다. | [가이드](modules/agents.md) |
| [agent/capture.go](<../../agent/capture.go>) | Norma 세션 이벤트를 ARTEX의 db.Activity 실행 기록으로 변환하는 경계이다. captureRun은 세션 생성·종료까지 소유하고, captureRunSession은 이미 열린 세션을 사용한다. | [가이드](modules/agents.md) |
| [agent/capture_approval_test.go](<../../agent/capture_approval_test.go>) | 가짜 Provider와 승인 훅을 연결해 tool_use 시작부터 승인 완료·활동 기록까지의 이벤트 순서를 검사한다. 실행 기록의 승인 요청 ID, 도구 인자, 결과가 같은 호출에 연결되는지 확인하는 회귀 테스트이다. | [가이드](modules/agents.md) |
| [agent/capture_usage_test.go](<../../agent/capture_usage_test.go>) | Provider 실패와 context 취소 직전까지 받은 usage가 최종 Activity에도 남는지 검사한다. 성공한 답변만 세면 실패한 모델 요청의 비용이 사라지는 문제를 막기 위한 테스트이다. | [가이드](modules/agents.md) |
| [agent/chat.go](<../../agent/chat.go>) | 일반 대화 화면과 사용자 정의 Agent를 실행하는 ChatAgent이다. Chat 호출 한 번마다 새 Norma Session을 만들되 sessionID의 transcript를 Resume하여 대화를 잇는다. | [가이드](modules/agents.md) |
| [agent/coldgraph.go](<../../agent/coldgraph.go>) | 탐색 그래프에서 당장 필요한 노드와 요약 가능한 노드를 구분하는 순수 알고리즘이다. cgNode/cgEdge는 DB 전체 객체 대신 ID·종류·상태·간선만 보유하는 계산용 뷰이다. | [가이드](modules/agents.md) |
| [agent/coldgraph_test.go](<../../agent/coldgraph_test.go>) | DB·모델 호출 없이 hot/cold 판단과 union-find 묶음을 검증한다. 살아 있는 한 분기가 조상을 보존하는지, 완전히 종료된 체인만 접히는지, 6회차 대기·같은 부모 아래 단독 노드·내용 버전 해시를 검사한다. | [가이드](modules/agents.md) |
| [agent/compaction.go](<../../agent/compaction.go>) | coldgraph의 후보 계산을 PostgreSQL 저장소와 LLM 요약 호출에 연결한다. 매 Planner 회차의 maintain은 회차·cold 시작 시점만 갱신하고 모델을 호출하지 않는다. | [가이드](modules/agents.md) |
| [agent/constraints.go](<../../agent/constraints.go>) | task_constraints의 allow/deny 문장을 역할 프롬프트에 붙이는 렌더러이다. 빈 제약이나 저장소 부재·조회 실패 때는 빈 문자열을 반환한다. | [가이드](modules/agents.md) |
| [agent/deferred.go](<../../agent/deferred.go>) | MCP 도구의 스키마를 처음부터 모두 보내지 않기 위한 프롬프트·해제 보조 코드이다. 전역 공개 도구 이름은 마지막 system 구간에 넣고 Skill 전용 이름은 그 구간에서 제외한다. | [가이드](modules/agents.md) |
| [agent/deferred_test.go](<../../agent/deferred_test.go>) | 지연 도구 이름의 system 구간과 캐시 경계, Skill 전용 이름 비노출, 과거 Skill 호출에서 UnlockSet 복원을 검증한다. 도구를 알리는 것과 호출 게이트를 여는 것이 별도 단계임을 보여 준다. | [가이드](modules/agents.md) |
| [agent/finding_recorder.go](<../../agent/finding_recorder.go>) | Agent와 증거 저장소 사이의 의존성을 작게 유지하는 FindingRecorder 인터페이스이다. Agent는 RecordFindingInput과 검증할 TrafficRef 목록만 전달하고 HTTP 본문 복사는 수행하지 않는다. | [가이드](modules/agents.md) |
| [agent/finding_recorder_test.go](<../../agent/finding_recorder_test.go>) | 선택적인 HTTP 증거가 없는 정상 취약점 기록과 기록기 실패의 원자적 계약을 검사한다. 잘못된 트래픽 참조나 기록 실패가 탐색 노드/독립 finding의 불완전한 성공으로 보이지 않아야 한다. | [가이드](modules/agents.md) |
| [agent/finding_workflow.go](<../../agent/finding_workflow.go>) | 최종 도구 목록에 취약점과 HTTP 증거를 넘기는 공통 계약을 적용한다. finding_id는 독립 findings 레코드 ID이고 finding_node_id는 exploration_nodes ID이다. | [가이드](modules/agents.md) |
| [agent/finding_workflow_test.go](<../../agent/finding_workflow_test.go>) | 여러 역할의 도구 조립에 동일한 finding 증거 계약이 적용되는지 검사한다. 기능을 끄고 다시 켜도 원본 schema와 바인딩이 손상되지 않는지, 설명 덮어쓰기와 안내가 함께 유지되는지 확인한다. | [가이드](modules/agents.md) |
| [agent/goals.go](<../../agent/goals.go>) | 사용자의 목표·설명에서 최종 산출 목표와 실행 제약, 명시된 대상 범위를 추출한다. DecomposeGoalsWithProvider는 별도 모델을 만들지 않고 호출자가 주는 Provider를 공유한다. | [가이드](modules/agents.md) |
| [agent/insert_assets_test.go](<../../agent/insert_assets_test.go>) | insert_assets를 실제 저장소와 연결하여 유형별 Upsert 결과와 파생 자산을 검사한다. 여러 IP의 하위 도메인, HTTP 기술 목록 병합, 비 HTTP 서비스, 혼합 배치 부분 성공, 중복 입력 및 잘못된 IP를 검증한다. | [가이드](modules/agents.md) |
| [agent/mainagent.go](<../../agent/mainagent.go>) | 작업 상세 화면에서 사람과 대화하며 작업을 관찰·조정하는 MainAgent이다. 자동으로 반복 계획을 세우는 루프는 Planner가 담당하며 MainAgent.Chat은 사용자 메시지 한 번을 처리한다. | [가이드](modules/agents.md) |
| [agent/noa.go](<../../agent/noa.go>) | 선택 기능인 Norma noa 컨텍스트 압축 어댑터를 세션 옵션에 연결한다. 설정 콜백은 매 run 시작 때 읽히며 nil 또는 false이면 기본 압축 경로를 그대로 둔다. | [가이드](modules/agents.md) |
| [agent/planner.go](<../../agent/planner.go>) | 서버 이벤트에 반응하여 작업의 다음 intent와 목표 달성 여부를 결정하는 Planner이다. Plan은 매번 최신 그래프 요약·발생 이벤트를 읽고 새 Norma Session으로 한 회차를 실행한다. | [가이드](modules/agents.md) |
| [agent/prompt.go](<../../agent/prompt.go>) | DB에서 편집한 역할 프롬프트와 코드 기본 프롬프트를 같은 Go template 방식으로 렌더링한다. PromptOverride가 유효한 값을 주면 우선 사용하고 실패하면 기본 템플릿을 다시 시도한다. | [가이드](modules/agents.md) |
| [agent/prompt_now_test.go](<../../agent/prompt_now_test.go>) | 사용자 정의 대화 프롬프트의 Now와 DataDir 변수 및 알 수 없는 변수의 기본값 복귀를 검사한다. DB에 저장한 템플릿과 코드 기본 템플릿이 같은 렌더러를 거친다는 계약을 확인한다. | [가이드](modules/agents.md) |
| [agent/prompt_test.go](<../../agent/prompt_test.go>) | 역할별 프롬프트 override, 빈값, 문법 오류, 지원하지 않는 변수에서의 복귀 동작을 검사한다. 전역 PromptOverride를 임시 교체하므로 테스트 종료 시 원래 훅을 복원하는 코드도 함께 읽는다. | [가이드](modules/agents.md) |
| [agent/promptcatalog.go](<../../agent/promptcatalog.go>) | 역할별 기본 모델 지시문과 서버 초기 DB 시딩용 목록을 모은다. auto는 플랫폼 조작, pentest는 독립 수행, reporter는 등록된 취약점 보고서 작성을 지시한다. | [가이드](modules/agents.md) |
| [agent/provider.go](<../../agent/provider.go>) | 모델 설정을 Norma Provider와 HTTP 전송 계층으로 연결한다. 환경 변수 또는 UI 입력을 읽어 Anthropic/OpenAI/OpenAI Responses 형식, URL, 모델, 키를 구성한다. | [가이드](modules/agents.md) |
| [agent/provider_capture_e2e_test.go](<../../agent/provider_capture_e2e_test.go>) | 로컬 HTTP 테스트 서버와 실제 Norma Provider를 함께 사용해 원본 요청/응답 캡처가 전송 계층까지 이어지는지 검사한다. 순수 래퍼 테스트에서 놓칠 수 있는 SDK 경계의 계약을 확인한다. | [가이드](modules/agents.md) |
| [agent/provider_capture_test.go](<../../agent/provider_capture_test.go>) | quotaAwareTransport가 본문을 손실 없이 캡처하는지 검사한다. SSE 200, 잔액 소진 429→402 정규화, 캡처 비활성화의 원래 응답 보존을 각각 확인한다. | [가이드](modules/agents.md) |
| [agent/provider_quota_test.go](<../../agent/provider_quota_test.go>) | 명시적 잔액 소진과 일시적 속도 제한·인증·서버 오류를 구별하는 문자열 분류를 검사한다. 429 본문 판별 이후 SDK가 불필요하게 같은 소진 계정에 재시도하지 않는지도 로컬 HTTP 서버로 검증한다. | [가이드](modules/agents.md) |
| [agent/provider_responses_test.go](<../../agent/provider_responses_test.go>) | openai-responses UI 설정이 Norma 형식과 /responses를 제거한 API base로 변환되는지 검사한다. 기본 모델/형식 식별이 OpenAI Chat Completions 경로와 혼동되지 않는지 확인한다. | [가이드](modules/agents.md) |
| [agent/proxyenv_test.go](<../../agent/proxyenv_test.go>) | 프록시가 없을 때 환경이 비어 있는지, SOCKS5 경로에 ALL_PROXY가 생기는지, MITM 기록일 때 각 언어 도구의 CA 변수가 붙는지 검사한다. 실제 외부 트래픽을 보내는 테스트가 아니다. | [가이드](modules/agents.md) |
| [agent/retester.go](<../../agent/retester.go>) | 재검증 전용 대화 Agent의 기본 프롬프트를 보관한다. 이미 등록된 한 취약점의 과거 증거와 제약을 읽고 현재 상태를 별도 회차로 확인하도록 지시한다. | [가이드](modules/agents.md) |
| [agent/review_context_test.go](<../../agent/review_context_test.go>) | Worker의 여러 도구 호출을 거치며 승인 심사 문맥이 올바르게 갱신되는지 검사한다. 현재 동작과 이미 수행한 짝지어진 도구 증거를 구별하고 상위 실행의 광범위한 목적이 심사 배경으로 잘못 이어지지 않는지 확인한다. | [가이드](modules/agents.md) |
| [agent/runinfo.go](<../../agent/runinfo.go>) | 도구 조립 시 해당 호출이 어느 작업·탐색·intent·대화에 속하는지 전달한다. AugmentTools의 입력에 모든 ID를 추가하는 대신 context 값으로 RunInfo를 주입한다. | [가이드](modules/agents.md) |
| [agent/session_header_test.go](<../../agent/session_header_test.go>) | HTTP 요청 context의 transcript 세션 ID를 사용자 지정 헤더에 전달하는지 검사한다. 헤더 이름 또는 세션 ID가 비어 있을 때 헤더를 생성하지 않는 경계도 확인한다. | [가이드](modules/agents.md) |
| [agent/side_questions.go](<../../agent/side_questions.go>) | 기존 Agent의 RunInfo를 /btw의 부모 리소스 식별자로 바꾸는 접착 코드이다. conv- 접두어가 있는 일반 대화와 작업·탐색·intent가 있는 실행을 각각 식별한다. | [가이드](modules/agents.md) |
| [agent/side_questions_test.go](<../../agent/side_questions_test.go>) | 실제 ChatAgent와 가짜 모델·로컬 파일 도구를 조합해 /btw 체크포인트를 검증한다. 주 도구 결과는 스냅샷에 포함되지만 독립 질문은 주 transcript와 활동 스트림을 변경하지 않아야 한다. | [가이드](modules/agents.md) |
| [agent/taskclock.go](<../../agent/taskclock.go>) | 서버의 작업 전체 마감 시각을 각 Planner/Worker run에 전달한다. DeadlineUnix가 0이면 작업 전체 시간 제한이 없고 Final은 최종 Planner 정리 회차를 뜻한다. | [가이드](modules/agents.md) |
| [agent/terminalreason.go](<../../agent/terminalreason.go>) | 모델이 최종 설명을 남기지 못한 경우에도 읽을 수 있는 실행 종료 기록을 만든다. runTrace는 마지막 도구의 ID·인자·시작 시간과 아직 결과가 안 왔는지를 보관한다. | [가이드](modules/agents.md) |
| [agent/terminalreason_test.go](<../../agent/terminalreason_test.go>) | 모든 Norma 종료 이유의 설명과 context 취소 원인 전파를 검사한다. 중단 전 부분 답변 보존, 아직 끝나지 않은 도구 식별, 이미 완료된 도구의 구분, 한국어를 포함한 rune 기준 잘림을 검증한다. | [가이드](modules/agents.md) |
| [agent/testmain_test.go](<../../agent/testmain_test.go>) | db.DSN으로 PostgreSQL 연결을 얻어 agent 테스트 전체 동안 advisory lock 7337741002를 잡는다. db/server 패키지와 테스트 행 DELETE 정리가 경쟁하는 것을 줄이기 위한 실행 직렬화이며 독립 테스트 DB에서 사용한다. | [가이드](modules/agents.md) |
| [agent/toolcatalog.go](<../../agent/toolcatalog.go>) | Go에 정의된 도구를 DB 관리 화면에서 편집·바인딩할 수 있는 카탈로그로 만든다. BuiltinToolSeeds는 역할별 기본 목록을 이름으로 합치며 동일 도구의 역할을 합집합으로 기록한다. | [가이드](modules/agents.md) |
| [agent/toolcatalog_test.go](<../../agent/toolcatalog_test.go>) | 도구 카탈로그에 기본 역할별 바인딩이 맞게 합쳐지는지 검사한다. 설명/schema 덮어쓰기는 원래 handler를 유지하고 누락·null·빈 문자열에만 scalar 기본값을 넣는지 확인한다. | [가이드](modules/agents.md) |
| [agent/tools.go](<../../agent/tools.go>) | LLM이 탐색 그래프를 읽고 쓰는 핵심 도메인 도구를 정의한다. ToolSet 하나가 한 run의 task·owner intent·콜백·쓰기 건수를 보유하며 DB 저장소를 감싼다. | [가이드](modules/agents.md) |
| [agent/tools_digest.go](<../../agent/tools_digest.go>) | 압축된 탐색 기록을 overview에 노출하고 필요할 때 원본으로 내려가는 읽기 계층이다. coldDigestsRecent는 멤버 중 가장 큰 ID를 최신성의 근사값으로 사용해 digest를 정렬한다. | [가이드](modules/agents.md) |
| [agent/tools_insert.go](<../../agent/tools_insert.go>) | 자산 입력과 기업·작업 범위를 LLM 도구 형태로 제공한다. insert_assets는 여섯 자산 유형을 섞은 배열을 받아 항목별 Upsert와 성공/오류 목록을 반환한다. | [가이드](modules/agents.md) |
| [agent/tools_nil_store_test.go](<../../agent/tools_nil_store_test.go>) | 작업 저장소가 없는 역할에 도메인 도구가 바인딩되어도 프로세스가 죽지 않는지 검사한다. task 필요 오류를 반환하고 예외적인 handler panic은 guardPanic이 도구 오류로 바꿔야 한다. | [가이드](modules/agents.md) |
| [agent/tools_overview_test.go](<../../agent/tools_overview_test.go>) | 여러 source 작업에 overview 텍스트 예산을 나눌 때 공정성과 UTF-8 경계를 검사한다. 잘린 문자열에 유효하지 않은 한국어/중국어 바이트가 남거나 총 예산을 초과하면 실패한다. | [가이드](modules/agents.md) |
| [agent/worker.go](<../../agent/worker.go>) | 이미 할당된 한 intent를 실제 도구로 수행하고 관찰·자산·취약점을 기록하는 Worker이다. intent를 가져오는 원자적 claim은 서버 엔진이 담당하며 Execute는 그 결과 노드를 입력받는다. | [가이드](modules/agents.md) |
| [agent/worker_intervention_test.go](<../../agent/worker_intervention_test.go>) | Worker 슬롯 대신 exploration/intent ID로 transcript 키가 안정되게 생성되는지 검사한다. 인간 개입 request ID 마커는 정상 user 메시지에서만 인식하여 다른 역할의 텍스트를 중복 요청으로 오인하지 않는다. | [가이드](modules/agents.md) |
| [agent/wrapup.go](<../../agent/wrapup.go>) | 에이전트가 예산을 다 썼을 때 사용할 수습 지시와 최대 회차를 구성한다. 역할별 코드 기본값에 DB의 비어 있지 않은 프롬프트/양수 회차 설정을 우선 적용한다. | [가이드](modules/agents.md) |

## cmd — 2개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [cmd/artex/main.go](<../../cmd/artex/main.go>) | 실행 파일의 시작과 종료 run은 플래그 해석 → 업데이트 부트스트랩 → PostgreSQL Manager → Server → HTTP 수신 순서로 시작한다. | [가이드](modules/server.md) |
| [cmd/artex/main_test.go](<../../cmd/artex/main_test.go>) | 프로세스 종료 원인의 회귀 테스트 신호 컨텍스트를 취소하고 shutdownContext 결과에서 AbortShutdown의 명명된 사유가 보존되는지 검사한다. | [가이드](modules/server.md) |

## config — 2개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [config/config.go](<../../config/config.go>) | 설정 경로와 PostgreSQL 연결 문자열 설정 파일 위치는 ARTEX_CONFIG → 현재 디렉터리의 config.json → 실행 파일 옆 config.json 순서로 찾는다. | [가이드](modules/server.md) |
| [config/config_test.go](<../../config/config_test.go>) | DB 설정 우선순위의 회귀 테스트 임시 config.json과 테스트 전용 환경 변수를 구성해 파일 필드 조합·환경 DSN 우선·누락 오류를 확인한다. | [가이드](modules/server.md) |

## db — 96개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [db/activity_page_test.go](<../../db/activity_page_test.go>) | 한국어 테스트 안내 대화 세그먼트별 activity 페이지와 노드 종류별 페이지 조회를 검증한다. | [가이드](modules/database.md) |
| [db/asset_dsl.go](<../../db/asset_dsl.go>) | 한국어 읽기 안내 자산 검색 문장을 토큰 → AND/OR 식 트리 → 매개변수화한 SQL WHERE로 바꾸는 작은 검색 언어 구현이다. | [가이드](modules/database.md) |
| [db/asset_intercept.go](<../../db/asset_intercept.go>) | 한국어 읽기 안내 전역 자산 차단 규칙의 영속화 계층이다. kind·pattern·note·enabled를 저장하고 화면에서 편집할 수 있게 CRUD를 제공한다. | [가이드](modules/database.md) |
| [db/asset_intercept_match.go](<../../db/asset_intercept_match.go>) | 한국어 읽기 안내 자산의 domain/IP/URL 후보와 전역·작업 규칙을 비교하는 순수 판정 로직 및 DB 연결 부분이다. | [가이드](modules/database.md) |
| [db/asset_intercept_match_test.go](<../../db/asset_intercept_match_test.go>) | 한국어 테스트 안내 도메인/CIDR/IP/URL 패턴의 일치, 비활성 규칙 제외, 차단/허용 우선순위를 표 기반으로 검사한다. | [가이드](modules/database.md) |
| [db/assets.go](<../../db/assets.go>) | 한국어 읽기 안내 여섯 자산 종류(root_domain, ip, subdomain, app, service, endpoint)의 전역 공유 저장소다. 서로 다른 종류의 열을 하나의 assets 테이블에 담는다. | [가이드](modules/database.md) |
| [db/assets_test.go](<../../db/assets_test.go>) | 한국어 테스트 안내 여섯 자산 종류의 upsert, 자연키 중복 제거, 부가 필드 병합, 회사 귀속, 작업/회사별 페이지 조회와 삭제를 검증한다. | [가이드](modules/database.md) |
| [db/chat_mentions.go](<../../db/chat_mentions.go>) | 한국어 읽기 안내 대화 입력의 @ 참조 선택기에 필요한 작업·자산·발견 등 후보를 검색한다. 종류는 ValidChatMentionKind의 허용 목록을 따른다. | [가이드](modules/database.md) |
| [db/commands.go](<../../db/commands.go>) | 한국어 읽기 안내 activity에 저장된 도구 호출을 명령 이력으로 조회하고, 별도의 llm_records에 모델 요청/응답 원문 기록을 보관한다. | [가이드](modules/database.md) |
| [db/companies.go](<../../db/companies.go>) | 한국어 읽기 안내 회사 레지스트리와 회사별 범위 규칙을 관리하고, 그 규칙으로 전역 자산의 company_id를 재계산한다. | [가이드](modules/database.md) |
| [db/companies_test.go](<../../db/companies_test.go>) | 한국어 테스트 안내 정규화 회사명 중복, 회사와 범위의 원자적 생성, 규칙 갱신 실패 롤백, 회사 귀속 재계산과 삭제를 검증한다. | [가이드](modules/database.md) |
| [db/company_lock_test.go](<../../db/company_lock_test.go>) | 한국어 테스트 안내 회사 범위 변경용 advisory lock 키가 스키마/테스트/증거 등 다른 기반 잠금과 충돌하지 않는지 검사한다. | [가이드](modules/database.md) |
| [db/company_scope.go](<../../db/company_scope.go>) | 한국어 읽기 안내 사용자가 입력한 회사 범위를 domain/IP/CIDR/ICP/keyword로 분류하고 정규화하는 파서다. | [가이드](modules/database.md) |
| [db/company_scope_consistency_test.go](<../../db/company_scope_consistency_test.go>) | 한국어 테스트 안내 범위 수정과 자산 upsert가 같은 잠금으로 조율되는지, 동일 우선순위에서 회사 ID로 결과가 안정되는지 검증한다. | [가이드](modules/database.md) |
| [db/company_scope_test.go](<../../db/company_scope_test.go>) | 한국어 테스트 안내 자동/명시적 범위 입력 파서와 domain/IP/CIDR/ICP/keyword 분류, ICP 귀속을 검증한다. | [가이드](modules/database.md) |
| [db/config.go](<../../db/config.go>) | 한국어 읽기 안내 LLM 프로필, 에이전트 설정, 프롬프트 버전, MCP 서버와 도구 캐시, Skill/MCP 가시성을 PostgreSQL에 저장한다. | [가이드](modules/database.md) |
| [db/config_max_tokens_test.go](<../../db/config_max_tokens_test.go>) | 한국어 테스트 안내 LLM 프로필의 max_tokens가 저장·조회·수정 후 같은 값으로 돌아오는지와 기본값 적용을 검증한다. | [가이드](modules/database.md) |
| [db/config_test.go](<../../db/config_test.go>) | 한국어 테스트 안내 프로필 failover 정렬, 취소된 삭제 요청, 에이전트/프롬프트/MCP/가시성 저장소의 기본 계약을 검증한다. | [가이드](modules/database.md) |
| [db/constants.go](<../../db/constants.go>) | 한국어 읽기 안내 탐색 그래프의 노드 종류·관계 이름·상태 이름을 공유하는 상수 모음이다. 문자열 자체는 DB CHECK 제약 및 API 계약과 맞물린다. | [가이드](modules/database.md) |
| [db/constraints.go](<../../db/constraints.go>) | 한국어 읽기 안내 한 exploration에 속한 사람이 읽을 수 있는 작업 제약을 task_constraints에 저장한다. 목록·추가·수정·삭제가 exploration_id로 제한된다. | [가이드](modules/database.md) |
| [db/conversation.go](<../../db/conversation.go>) | 한국어 읽기 안내 독립 대화의 제목·에이전트·모델 설정·고정 상태와 conversation_activities를 관리한다. 작업 탐색의 activity와 저장 공간이 구분된다. | [가이드](modules/database.md) |
| [db/conversation_pin_test.go](<../../db/conversation_pin_test.go>) | 한국어 테스트 안내 대화 고정/해제와 PATCH 후 목록 순서를 검증한다. | [가이드](modules/database.md) |
| [db/customtool_test.go](<../../db/customtool_test.go>) | 한국어 테스트 안내 Python/HTTP 등의 사용자 정의 도구 레코드 생성·읽기·수정·삭제를 검증한다. | [가이드](modules/database.md) |
| [db/db.go](<../../db/db.go>) | 한국어 읽기 안내 PostgreSQL 연결과 초기화의 진입점이다. DSN → 연결 확인 → 스키마 적용 → 내장 설정 초기값 등록 순서로 시작한다. | [가이드](modules/database.md) |
| [db/db_test.go](<../../db/db_test.go>) | 한국어 테스트 안내 스키마 실행이 PostgreSQL deadlock(40P01)만 재시도하는지와 Open의 내장 데이터 등록/재호출 멱등성을 검증한다. | [가이드](modules/database.md) |
| [db/digest.go](<../../db/digest.go>) | 한국어 읽기 안내 탐색 그래프의 오래된 부분을 요약하는 compactor가 사용하는 영속 상태를 담당한다. 요약문 생성 자체는 agent 쪽에서 수행한다. | [가이드](modules/database.md) |
| [db/exploration.go](<../../db/exploration.go>) | 한국어 읽기 안내 한 작업의 탐색 그래프와 실행 활동 장부를 읽고 쓰는 중심 파일이다. Node는 추론/계획 상태, Edge는 노드 간 관계, Anchor는 전역 자산 참조다. | [가이드](modules/database.md) |
| [db/exploration_sources.go](<../../db/exploration_sources.go>) | 한국어 읽기 안내 현재 작업이 직접 연결한 source 작업의 기록을 읽기 전용 문맥으로 합치는 조회 계층이다. source의 source까지 재귀로 확대하지 않는다. | [가이드](modules/database.md) |
| [db/exploration_sources_test.go](<../../db/exploration_sources_test.go>) | 한국어 테스트 안내 직접 source의 읽기 전용 노드·fact·종료된 worker 이력·앵커를 합치는 조회 계약을 검증한다. | [가이드](modules/database.md) |
| [db/exploration_test.go](<../../db/exploration_test.go>) | 한국어 테스트 안내 탐색 노드/간선/앵커의 생성·조회·상태 변경, 선점 경쟁, activity 저장과 계보 조회를 검증한다. | [가이드](modules/database.md) |
| [db/exploration_tokens_test.go](<../../db/exploration_tokens_test.go>) | 한국어 테스트 안내 작업 전체·worker·UI 세션의 토큰 합계가 종료 result 활동에서 일관되게 집계되는지 검증한다. | [가이드](modules/database.md) |
| [db/finding_assets.go](<../../db/finding_assets.go>) | 한국어 읽기 안내 발견 목록의 자산별 탐색 트리를 만드는 조회/조립 코드다. finding.asset_ids로 출발해 실제 자산과 필요한 상위 호스트·회사를 보완한다. | [가이드](modules/database.md) |
| [db/finding_assets_test.go](<../../db/finding_assets_test.go>) | 한국어 테스트 안내 발견 필터에 따른 자산 트리와 상위 호스트/회사 연결, 심각도 집계 및 하위 범위 필터를 검증한다. | [가이드](modules/database.md) |
| [db/finding_retests.go](<../../db/finding_retests.go>) | 한국어 읽기 안내 사용자가 발견 재검증을 요청했을 때 재검증 대화와 원본 스냅샷을 만들고 결과를 봉인하는 저장 계층이다. | [가이드](modules/database.md) |
| [db/finding_retests_test.go](<../../db/finding_retests_test.go>) | 한국어 테스트 안내 동시 재검증 요청 중복 억제, 원본 스냅샷/대화 생성, 결과 판정과 완료 상태의 조합을 검증한다. | [가이드](modules/database.md) |
| [db/finding_traffic.go](<../../db/finding_traffic.go>) | 한국어 읽기 안내 발견과 HTTP 증거 스냅샷의 연결, 증거 편집 버전, 보고서가 참조한 버전을 관리한다. 실제 본문 파일 복사/해시 검사는 evidence 패키지와 협력한다. | [가이드](modules/database.md) |
| [db/finding_traffic_archive.go](<../../db/finding_traffic_archive.go>) | 한국어 읽기 안내 작업 아카이브에 포함된 발견 증거 스냅샷을 읽고, 복원 때 스냅샷과 발견 연결을 다시 구성한다. | [가이드](modules/database.md) |
| [db/finding_traffic_archive_test.go](<../../db/finding_traffic_archive_test.go>) | 한국어 테스트 안내 증거용 advisory lock 키 충돌 방지와 구형 아카이브에서 없는 발견 버전/증거 연결의 기본값 복원을 검증한다. | [가이드](modules/database.md) |
| [db/findings.go](<../../db/findings.go>) | 한국어 읽기 안내 독립 findings 테이블의 목록·필터·통계·편집·삭제·내보내기를 담당한다. 작업 그래프의 finding 노드와 node_id로 연결된다. | [가이드](modules/database.md) |
| [db/findings_test.go](<../../db/findings_test.go>) | 한국어 테스트 안내 발견의 별도 목록 테이블과 그래프 노드 연결, 상태·심각도·필터·그룹·내보내기·후속 intent 생성을 검증한다. | [가이드](modules/database.md) |
| [db/intent_admission.go](<../../db/intent_admission.go>) | 한국어 읽기 안내 추가 intent를 만들었으나 작업 실행 허가 단계가 실패한 경우 저장 결과를 보상하는 작은 트랜잭션이다. | [가이드](modules/database.md) |
| [db/intent_control_test.go](<../../db/intent_control_test.go>) | 한국어 테스트 안내 사람의 pause/resume/stop과 worker의 완료가 경쟁할 때 허용된 상태 전이만 성공하는지 검증한다. | [가이드](modules/database.md) |
| [db/intent_delete_test.go](<../../db/intent_delete_test.go>) | 한국어 테스트 안내 intent의 soft delete, 삭제 사유, 관련 후손/이력 처리와 재조회 시 노출 범위를 검증한다. | [가이드](modules/database.md) |
| [db/intercept.go](<../../db/intercept.go>) | 한국어 읽기 안내 도구 호출 승인 규칙과 승인 대기/결정 이력을 보관한다. 자산 주소 규칙을 다루는 asset_intercept와 다른 기능이다. | [가이드](modules/database.md) |
| [db/intercept_detail.go](<../../db/intercept_detail.go>) | 한국어 읽기 안내 승인 순간에 수집한 제한된 세션 사건과 도구 ID를 audit JSON으로 저장/조회한다. 기록된 맥락이며 모델 내부 추론을 복원하지 않는다. | [가이드](modules/database.md) |
| [db/intercept_detail_test.go](<../../db/intercept_detail_test.go>) | 한국어 테스트 안내 승인 당시 audit의 저장/상세 조회와 사람이 결정한 뒤 타임아웃/중복 결정이 덮어쓰지 않는 조건을 검증한다. | [가이드](modules/database.md) |
| [db/intercept_execution.go](<../../db/intercept_execution.go>) | 한국어 읽기 안내 승인 이력에서 원래 도구 실행 위치로 이동할 수 있도록 activity/conversation_activities의 정확한 호출을 찾는다. | [가이드](modules/database.md) |
| [db/intercept_execution_test.go](<../../db/intercept_execution_test.go>) | 한국어 테스트 안내 승인에서 원본 tool_use/tool_result 쌍으로 이동할 때 ID·worker·intent·main 세그먼트의 정확한 연결을 검증한다. | [가이드](modules/database.md) |
| [db/intercept_filter_test.go](<../../db/intercept_filter_test.go>) | 한국어 테스트 안내 승인 목록의 상태/종류/대화/작업/검색 필터와 페이지 건수를 검증한다. | [가이드](modules/database.md) |
| [db/intercept_seed_test.go](<../../db/intercept_seed_test.go>) | 한국어 테스트 안내 내장 삭제 경로 정규식이 tool_input JSON 형태에서 삭제 API를 탐지하고 비슷한 일반 단어는 통과시키는지 검증한다. | [가이드](modules/database.md) |
| [db/jsonb_clean_test.go](<../../db/jsonb_clean_test.go>) | 한국어 테스트 안내 PostgreSQL JSONB가 거부하는 실제 NUL escape만 제거하고 이스케이프된 역슬래시의 문자 표현은 유지하는지 검증한다. | [가이드](modules/database.md) |
| [db/llm_records_migrate_test.go](<../../db/llm_records_migrate_test.go>) | 한국어 테스트 안내 예전 llm_records 테이블을 현재 기록 형태로 보완하는 마이그레이션의 멱등성을 검증한다. | [가이드](modules/database.md) |
| [db/llm_usage.go](<../../db/llm_usage.go>) | 한국어 읽기 안내 모델 호출별 토큰 사용량을 llm_usage에 누적하는 계량 장부와 집계 쿼리다. 원문 요청/응답을 저장하는 llm_records와 분리된다. | [가이드](modules/database.md) |
| [db/llmhealth.go](<../../db/llmhealth.go>) | 한국어 읽기 안내 LLM 프로필의 일시적 장애와 냉각 종료 시각을 저장해 프로세스 재시작 뒤에도 회로 차단 상태를 이어간다. | [가이드](modules/database.md) |
| [db/llmretry.go](<../../db/llmretry.go>) | 한국어 읽기 안내 전역 재시도 정책과 프로필별 부분 재정의를 표현한다. 횟수 0은 미설정, -1은 해당 재시도 비활성, 양수는 명시한 횟수다. | [가이드](modules/database.md) |
| [db/llmretry_test.go](<../../db/llmretry_test.go>) | 한국어 테스트 안내 프로필별 재시도 설정 및 전역 정책의 저장/조회, 값의 범위 제한, 미설정 기본값을 검증한다. | [가이드](modules/database.md) |
| [db/logs.go](<../../db/logs.go>) | 한국어 읽기 안내 운영 로그의 추가·최근 조회·이전 페이지 조회를 담당한다. 작업별 도구 활동과 달리 서버 전체 운용 사건을 담는 로그다. | [가이드](modules/database.md) |
| [db/nkey.go](<../../db/nkey.go>) | 한국어 읽기 안내 도메인·IP·URL·파라미터의 정규화와 과거 그래프 자연키를 만드는 보조 함수 모음이다. | [가이드](modules/database.md) |
| [db/notification.go](<../../db/notification.go>) | 한국어 읽기 안내 알림 채널 설정, 발견/상태 변경 사건, 채널별 발송 작업으로의 분배와 상태 통계를 저장한다. | [가이드](modules/database.md) |
| [db/notification_delivery.go](<../../db/notification_delivery.go>) | 한국어 읽기 안내 알림 전달 작업의 선점·임대·재시도·완료·이력 조회를 관리한다. 네트워크 전송 자체는 notify 패키지의 dispatcher가 담당한다. | [가이드](modules/database.md) |
| [db/notification_test.go](<../../db/notification_test.go>) | 한국어 테스트 안내 알림 채널 CRUD, 사건 분배, 실시간/요약 발송 선점, 임대·재시도·비활성 처리·통계를 검증한다. | [가이드](modules/database.md) |
| [db/schema.sql](<../../db/schema.sql>) | 한국어 스키마 안내 db.Open에서 매 시작마다 적용하는 멱등 스키마와 기존 설치 보정이다. | [가이드](modules/database.md) |
| [db/settings.go](<../../db/settings.go>) | 한국어 읽기 안내 settings 테이블의 문자열 key/value를 읽고 쓰는 공통 저장 API다. 기능별 세부 설정은 각 호출자가 직렬화해 저장한다. | [가이드](modules/database.md) |
| [db/side_questions.go](<../../db/side_questions.go>) | 한국어 읽기 안내 본 작업과 별도로 묻는 /btw 질문의 부모 스냅샷·요청·답변·요약 메모리를 저장한다. | [가이드](modules/database.md) |
| [db/side_questions_test.go](<../../db/side_questions_test.go>) | 한국어 테스트 안내 /btw 질문의 client ID 중복 억제, 기록 페이지, 재시작 복구, 메모리 체크포인트와 기록 초기화를 검증한다. | [가이드](modules/database.md) |
| [db/skill_usage.go](<../../db/skill_usage.go>) | 한국어 읽기 안내 Skill 호출의 성공/누락 여부와 에이전트·작업·탐색 식별자를 기록하는 사용 장부다. | [가이드](modules/database.md) |
| [db/skill_usage_test.go](<../../db/skill_usage_test.go>) | 한국어 테스트 안내 Skill 호출 성공/누락/에이전트/작업별 집계 및 최근 호출 목록을 검증한다. | [가이드](modules/database.md) |
| [db/task_archive_aggregate_stats.go](<../../db/task_archive_aggregate_stats.go>) | 한국어 읽기 안내 활성 테이블에서 제거된 아카이브 작업의 미리 계산된 통계를 읽어 전체 통계에 합친다. | [가이드](modules/database.md) |
| [db/task_archives.go](<../../db/task_archives.go>) | 한국어 읽기 안내 작업 아카이브의 상태 머신·작업 큐·형식 버전·DB 스냅샷 추출을 담당한다. 압축 파일 입출력은 상위 아카이브 서비스가 수행한다. | [가이드](modules/database.md) |
| [db/task_archives_restore.go](<../../db/task_archives_restore.go>) | 한국어 읽기 안내 검증된 아카이브 파일이 준비된 뒤 활성 DB를 축소하고, 복원 시 참조 순서에 맞춰 데이터를 되돌리는 트랜잭션 코드다. | [가이드](modules/database.md) |
| [db/task_archives_test.go](<../../db/task_archives_test.go>) | 한국어 테스트 안내 JSON 행 스트리밍, 지원 형식, 작업 아카이브→활성 정리→복원 왕복과 집계 유지·의존 관계·중단 복구를 검증한다. | [가이드](modules/database.md) |
| [db/task_assets.go](<../../db/task_assets.go>) | 한국어 읽기 안내 전역 자산을 특정 작업에 등록·연결·연결 해제하고 연결의 출처 설명을 기록한다. | [가이드](modules/database.md) |
| [db/task_assets_context.go](<../../db/task_assets_context.go>) | 한국어 읽기 안내 현재 작업과 직접 연결한 source 작업을 합쳐 범위·커버리지·미검사 자산·트래픽 검색용 호스트를 조회한다. | [가이드](modules/database.md) |
| [db/task_assets_test.go](<../../db/task_assets_test.go>) | 한국어 테스트 안내 범위 등록의 원자성, 기존 전역 자산 attach/detach, 출처 설명과 직접 source의 intent 대상 조회를 검증한다. | [가이드](modules/database.md) |
| [db/task_categories.go](<../../db/task_categories.go>) | 한국어 읽기 안내 작업 분류의 생성·이름 변경·삭제·단일/일괄 배정을 처리한다. 정규화된 이름의 고유 제약 위반을 도메인 오류로 바꾼다. | [가이드](modules/database.md) |
| [db/task_categories_test.go](<../../db/task_categories_test.go>) | 한국어 테스트 안내 분류 CRUD, 작업 생성 시 분류 검증, 단일/일괄 분류 배정과 잘못된 입력의 전체 롤백을 검증한다. | [가이드](modules/database.md) |
| [db/task_context.go](<../../db/task_context.go>) | 한국어 읽기 안내 작업의 직접 source 관계·회사 범위·순서 있는 LLM 체인과 현재 프로필 커서를 읽고 갱신한다. | [가이드](modules/database.md) |
| [db/task_context_lock_test.go](<../../db/task_context_lock_test.go>) | 한국어 테스트 안내 작업/에이전트/대화의 모델 프로필 참조 갱신과 프로필 삭제 사이의 잠금 순서를 실제 병렬 연결로 검증한다. | [가이드](modules/database.md) |
| [db/task_context_unit_test.go](<../../db/task_context_unit_test.go>) | 한국어 테스트 안내 UTF-8 문자열을 바이트 제한에 맞춰 자를 때 한글 같은 다중 바이트 문자가 중간에서 깨지지 않는지 검증한다. | [가이드](modules/database.md) |
| [db/task_delete_concurrency_test.go](<../../db/task_delete_concurrency_test.go>) | 한국어 테스트 안내 작업 삭제용 트래픽 호스트 선정부터 DB 삭제 확정까지 자산/앵커 쓰기를 같은 트랜잭션으로 조율하는지 검증한다. | [가이드](modules/database.md) |
| [db/task_intercept.go](<../../db/task_intercept.go>) | 한국어 읽기 안내 한 작업에 한정된 자산 allow/block 규칙의 CRUD와 생성 시 일괄 등록을 담당한다. | [가이드](modules/database.md) |
| [db/task_list_performance_test.go](<../../db/task_list_performance_test.go>) | 한국어 테스트 안내 작업 목록이 LLM 체인/source/회사 정보를 작업별 N+1 쿼리 없이 일괄 보완하는지 검증한다. | [가이드](modules/database.md) |
| [db/task_metadata_test.go](<../../db/task_metadata_test.go>) | 한국어 테스트 안내 작업 고정/해제·이름 수정·목록 순서와 대화 일괄 삭제의 실제 삭제 ID 반환을 검증한다. | [가이드](modules/database.md) |
| [db/task_queue_test.go](<../../db/task_queue_test.go>) | 한국어 테스트 안내 동시 실행 제한으로 큐에 들어간 작업의 bootstrap/resume 모드와 FIFO 시각이 반복 등록 후에도 보존되는지 검증한다. | [가이드](modules/database.md) |
| [db/task_scope.go](<../../db/task_scope.go>) | 한국어 읽기 안내 task_scope의 등록/삭제와 자산 포함 관계, 검사 커버리지 및 화면용 자산 그래프를 계산한다. | [가이드](modules/database.md) |
| [db/task_scope_test.go](<../../db/task_scope_test.go>) | 한국어 테스트 안내 host:port, IP:port, IPv6 표기의 포트를 제거하는 stripHostPort 경계 조건을 검증한다. | [가이드](modules/database.md) |
| [db/task_source_limit_test.go](<../../db/task_source_limit_test.go>) | 한국어 테스트 안내 최대 직접 source 수 및 회사 수, 회사 ID 양수 검증·순서 보존 중복 제거를 검사한다. | [가이드](modules/database.md) |
| [db/task_templates.go](<../../db/task_templates.go>) | 한국어 읽기 안내 재사용할 작업 템플릿의 이름·본문·옵션·자산 규칙을 저장하고 부분 갱신을 처리한다. | [가이드](modules/database.md) |
| [db/task_templates_test.go](<../../db/task_templates_test.go>) | 한국어 테스트 안내 템플릿 CRUD와 정규화 이름 중복 처리, 서로 다른 필드의 동시 PATCH 결합을 검증한다. | [가이드](modules/database.md) |
| [db/tasks.go](<../../db/tasks.go>) | 한국어 읽기 안내 작업 레지스트리와 exploration의 1:1 생명주기를 관리한다. 생성은 origin fact·source 관계·회사 범위·모델 체인·규칙을 한 트랜잭션에 묶는다. | [가이드](modules/database.md) |
| [db/tasks_test.go](<../../db/tasks_test.go>) | 한국어 테스트 안내 작업/탐색의 생성과 생명주기, cascade 삭제, 직접 source 관계, LLM failover 체인, 회사 범위·목록 정렬을 검증한다. | [가이드](modules/database.md) |
| [db/testmain_test.go](<../../db/testmain_test.go>) | 한국어 테스트 안내 db 테스트 묶음의 진입점이다. agent/server 테스트와 같은 advisory lock을 사용해 공유 테스트 DB의 청소 경쟁을 줄인다. | [가이드](modules/database.md) |
| [db/tool_usage.go](<../../db/tool_usage.go>) | 한국어 읽기 안내 도구 실행 시도마다 도구명·에이전트·작업·결과를 기록하는 가벼운 사용 장부다. | [가이드](modules/database.md) |
| [db/tool_usage_test.go](<../../db/tool_usage_test.go>) | 한국어 테스트 안내 도구 사용 장부의 기록과 도구별 호출 횟수를 검증한다. | [가이드](modules/database.md) |
| [db/tools.go](<../../db/tools.go>) | 한국어 읽기 안내 도구 설명·JSON Schema·허용 에이전트 목록·사용자 정의 도구의 실행 설정을 보관한다. | [가이드](modules/database.md) |
| [db/triggers.go](<../../db/triggers.go>) | 한국어 읽기 안내 사용자 정의 에이전트의 이벤트/주기 트리거와 스케줄러 커서를 영속화한다. | [가이드](modules/database.md) |

## docs — 1개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [docs/漏洞流量证据.md](<../../docs/漏洞流量证据.md>) | 원본 증거 기능 문서. 한국어판은 traffic-evidence.md에 있습니다. | [가이드](traffic-evidence.md) |

## enrich — 1개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [enrich/enrich.go](<../../enrich/enrich.go>) | enrich/enrich.go AI 호출 없이 DNS 및 HTTP 메타데이터를 보충할 수 있는 비동기 작업 풀이다. | [가이드](modules/services.md) |

## evidence — 3개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [evidence/store.go](<../../evidence/store.go>) | evidence/store.go finding에 연결된 HTTP 증거를 원시 traffic 정리와 독립적으로 보존한다. | [가이드](modules/services.md) |
| [evidence/store_test.go](<../../evidence/store_test.go>) | evidence/store_test.go PostgreSQL·임시 traffic·독립 evidence 디렉터리를 함께 사용하는 증거 통합 테스트다. | [가이드](modules/services.md) |
| [evidence/testmain_test.go](<../../evidence/testmain_test.go>) | evidence/testmain_test.go 증거 통합 테스트 묶음의 PostgreSQL 초기화와 공통 suite advisory lock을 관리한다. | [가이드](modules/services.md) |

## guard — 2개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [guard/guard.go](<../../guard/guard.go>) | guard/guard.go Norma의 도구 실행 전·후 hook을 ARTEX 승인 규칙 및 감사 기록에 연결한다. | [가이드](modules/services.md) |
| [guard/guard_test.go](<../../guard/guard_test.go>) | guard/guard_test.go 인터셉터 없는 Guard는 명령 문자열을 차단하지 않으면서 감사는 남긴다는 현재 계약을 검증한다. | [가이드](modules/services.md) |

## intercept — 7개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [intercept/intercept.go](<../../intercept/intercept.go>) | intercept/intercept.go 도구 호출 승인 상태를 DB 규칙 → 선택적 LLM 판정 → 필요 시 사람의 결정 순서로 처리한다. | [가이드](modules/services.md) |
| [intercept/prompt.go](<../../intercept/prompt.go>) | intercept/prompt.go LLM 승인 심사기의 기본 정책·입력 경계·출력 JSON 계약과 엄격한 응답 파서를 담는다. | [가이드](modules/services.md) |
| [intercept/prompt_test.go](<../../intercept/prompt_test.go>) | intercept/prompt_test.go LLM 심사 응답의 JSON 계약을 외부 모델 호출 없이 검증한다. | [가이드](modules/services.md) |
| [intercept/review_context.go](<../../intercept/review_context.go>) | intercept/review_context.go 모델 심사기에 보낼 현재 도구 호출의 입력 봉투를 만든다. | [가이드](modules/services.md) |
| [intercept/review_context_test.go](<../../intercept/review_context_test.go>) | intercept/review_context_test.go 심사기에 보낼 현재 호출 입력이 실행 감사 이력과 분리되는지 검사한다. | [가이드](modules/services.md) |
| [intercept/trace.go](<../../intercept/trace.go>) | intercept/trace.go 도구 시작·승인·완료 이벤트를 같은 실행 ID에 연결하여 감사의 출처를 보존한다. | [가이드](modules/services.md) |
| [intercept/trace_test.go](<../../intercept/trace_test.go>) | intercept/trace_test.go 한 Prompt 내 도구 시작·승인·완료 연결과 감사 문맥 한도를 검증한다. | [가이드](modules/services.md) |

## llmpool — 4개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [llmpool/health.go](<../../llmpool/health.go>) | 프로필 ID별 연속 실패·차단 횟수·냉각 종료 시각을 공유하는 회로 차단기이다. 인증/잔액/모델 설정 등 확정적 실패는 즉시, 일시적 실패는 기본 3회 후 차단한다. | [가이드](modules/agents.md) |
| [llmpool/health_policy_test.go](<../../llmpool/health_policy_test.go>) | 운영자 설정이 회로 차단기의 임계값과 냉각 시간을 바꾸는지 검사한다. 음수 임계값은 일시적 실패 차단만 끄고 확정적 실패는 유지하며 0은 기본 정책을 유지해야 한다. | [가이드](modules/agents.md) |
| [llmpool/pool.go](<../../llmpool/pool.go>) | 여러 LLM 프로필을 하나의 llm.Provider로 보이게 하는 장애 전환 장식자이다. 호출자는 이미 우선순위 순으로 정렬한 Member 목록을 제공하고 Pool은 건강 상태·문맥 크기로 후보를 거른다. | [가이드](modules/agents.md) |
| [llmpool/pool_test.go](<../../llmpool/pool_test.go>) | 가짜 Provider로 모델 장애 전환, 냉각 사다리, 성공 시 상태 초기화, 같은 순위 순환, 작은 창 제외를 검사한다. 첫 출력 뒤나 취소 후에는 다른 모델에 재전송하지 않는다는 가장 중요한 경계도 확인한다. | [가이드](modules/agents.md) |

## llmrec — 4개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [llmrec/capture.go](<../../llmrec/capture.go>) | 한 논리적 모델 요청에서 발생한 실제 HTTP 본문을 별도로 수집한다. 상위 Recorder가 context에 Capture를 넣으면 agent.quotaAwareTransport가 요청과 응답을 채운다. | [가이드](modules/agents.md) |
| [llmrec/capture_test.go](<../../llmrec/capture_test.go>) | nil 캡처의 무동작, 단일 응답 원문 보존, 재시도 응답 순서·상태 보존, HTTP가 없었던 호출의 빈 캡처를 검사한다. 원문 바이트와 재구성 JSON을 구별하는 작은 단위 테스트이다. | [가이드](modules/agents.md) |
| [llmrec/llmrec.go](<../../llmrec/llmrec.go>) | Provider의 스트림/완성 호출에 사용량 및 선택적 원문 기록을 추가한다. 가벼운 llm_usage 계측은 항상 수행하고 무거운 요청/응답 본문은 enabled 설정으로 제어한다. | [가이드](modules/agents.md) |
| [llmrec/llmrec_test.go](<../../llmrec/llmrec_test.go>) | 명시적 task ID와 exploration 기반 세션 ID의 구분, Complete 전달, 독립 질문의 사용량 귀속을 검사한다. 스트림 소비자가 일찍 중지해도 usage가 두 번 기록되거나 사라지지 않는지 확인한다. | [가이드](modules/agents.md) |

## mcphttp — 2개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [mcphttp/client.go](<../../mcphttp/client.go>) | mcphttp/client.go 원격 MCP 서버의 HTTP 또는 이전 SSE 전송을 Norma CoreTool 인터페이스로 맞추는 어댑터다. | [가이드](modules/services.md) |
| [mcphttp/client_sse_test.go](<../../mcphttp/client_sse_test.go>) | mcphttp/client_sse_test.go 로컬 httptest 서버로 이전 MCP SSE 전송의 발견·초기화·목록·호출을 연결해 검증한다. | [가이드](modules/services.md) |

## notify — 26개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [notify/channel.go](<../../notify/channel.go>) | notify/channel.go 6개 알림 채널을 하나의 무상태 Channel 인터페이스와 명시적 registry로 묶는다. | [가이드](modules/services.md) |
| [notify/channels_test.go](<../../notify/channels_test.go>) | notify/channels_test.go 로컬 HTTP 수신자와 메일 문자열 검증으로 6개 채널의 메시지·업무 오류·필수 설정 계약을 확인한다. | [가이드](modules/services.md) |
| [notify/dingtalk.go](<../../notify/dingtalk.go>) | notify/dingtalk.go DingTalk 봇에 Markdown 또는 상세 링크 버튼이 있는 ActionCard를 전송한다. | [가이드](modules/services.md) |
| [notify/email.go](<../../notify/email.go>) | notify/email.go SMTP 연결·선택적 인증·MAIL/RCPT/DATA 순서로 HTML 메일을 전송한다. | [가이드](modules/services.md) |
| [notify/email_test.go](<../../notify/email_test.go>) | notify/email_test.go 최소 SMTP 서버를 로컬에 구현하여 인증·발신자·수신자·DATA·종료의 실제 프로토콜 순서를 검증한다. | [가이드](modules/services.md) |
| [notify/event.go](<../../notify/event.go>) | notify/event.go DB 이벤트 스냅샷과 채널 렌더링용 메시지의 데이터 계약을 정의한다. | [가이드](modules/services.md) |
| [notify/feishu.go](<../../notify/feishu.go>) | notify/feishu.go Feishu/Lark 봇의 색상 있는 interactive 카드와 선택적 서명을 구성한다. | [가이드](modules/services.md) |
| [notify/filter.go](<../../notify/filter.go>) | notify/filter.go 채널별 심각도·작업·자산·분류 키워드와 상태 변경 수신 여부를 판별한다. | [가이드](modules/services.md) |
| [notify/filter_test.go](<../../notify/filter_test.go>) | notify/filter_test.go DB 없이 이벤트 스냅샷과 필터만으로 수신 선택 규칙을 검증한다. | [가이드](modules/services.md) |
| [notify/html.go](<../../notify/html.go>) | notify/html.go 메일용 HTML 본문을 간단한 인라인 스타일로 만든다. | [가이드](modules/services.md) |
| [notify/http.go](<../../notify/http.go>) | notify/http.go HTTP 알림의 연결·응답 제한·재시도 분류·URL 자격 증명 노출 방지를 공통 처리한다. | [가이드](modules/services.md) |
| [notify/http_test.go](<../../notify/http_test.go>) | notify/http_test.go 로컬 가짜 HTTP 수신자로 공통 전송층의 상태 코드 분류·진단 길이 제한을 검사한다. | [가이드](modules/services.md) |
| [notify/markdown.go](<../../notify/markdown.go>) | notify/markdown.go DingTalk·WeCom과 카드의 일부 본문에서 사용하는 Markdown 계열 렌더러다. | [가이드](modules/services.md) |
| [notify/mask.go](<../../notify/mask.go>) | notify/mask.go 채널 설정을 브라우저에 보여 줄 때 비밀 값을 마스킹하고 PATCH 시의 유지·교체·삭제 의미를 관리한다. | [가이드](modules/services.md) |
| [notify/mask_test.go](<../../notify/mask_test.go>) | notify/mask_test.go 비밀의 API 표시와 부분 설정 갱신의 유지·삭제·대상 변경 규칙을 검증한다. | [가이드](modules/services.md) |
| [notify/notify.go](<../../notify/notify.go>) | notify/notify.go 발견 알림의 공통 채널 ID·이벤트 ID·심각도/처리 상태 표시 규칙이다. | [가이드](modules/services.md) |
| [notify/pack_test.go](<../../notify/pack_test.go>) | notify/pack_test.go 메시지 길이 한도에서 앞부분 항목만 보냈을 때 kept가 실제 포함 수와 일치하는지 검증한다. | [가이드](modules/services.md) |
| [notify/redact_test.go](<../../notify/redact_test.go>) | notify/redact_test.go 채널별 URL에 들어 있는 테스트 토큰이 오류 경로로 노출되지 않는지 검증한다. | [가이드](modules/services.md) |
| [notify/render.go](<../../notify/render.go>) | notify/render.go 채널별 메시지 길이 예산·UTF-8 절단·HTML 경계·묶음 포장을 공통 처리한다. | [가이드](modules/services.md) |
| [notify/render_test.go](<../../notify/render_test.go>) | notify/render_test.go 메시지 절단의 모든 바이트/문자 경계와 HTML 끝부분을 직접 검사한다. | [가이드](modules/services.md) |
| [notify/sign_test.go](<../../notify/sign_test.go>) | notify/sign_test.go 서명 함수 결과를 구현과 독립적으로 미리 계산한 기준 문자열과 대조한다. | [가이드](modules/services.md) |
| [notify/ssrf_test.go](<../../notify/ssrf_test.go>) | notify/ssrf_test.go 로컬 가짜 수신자로 기본 연결 IP 제한과 명시적 해제 설정을 검증한다. | [가이드](modules/services.md) |
| [notify/telegram.go](<../../notify/telegram.go>) | notify/telegram.go Telegram Bot API의 sendMessage에 HTML 형식 알림을 전송한다. | [가이드](modules/services.md) |
| [notify/webhook.go](<../../notify/webhook.go>) | notify/webhook.go 사용자가 정한 URL·HTTP 메서드·헤더·JSON 템플릿으로 외부 수신 시스템을 연결하는 범용 채널이다. | [가이드](modules/services.md) |
| [notify/webhook_template_test.go](<../../notify/webhook_template_test.go>) | notify/webhook_template_test.go 사용자 Webhook 템플릿이 접근할 수 있는 데이터·함수 경계를 검증한다. | [가이드](modules/services.md) |
| [notify/wecom.go](<../../notify/wecom.go>) | notify/wecom.go WeCom 그룹 봇의 Markdown 채널이다. | [가이드](modules/services.md) |

## report — 2개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [report/findings.go](<../../report/findings.go>) | report/findings.go findings 테이블의 발견을 일괄/단일 Markdown과 CSV로 내보내는 렌더러다. | [가이드](modules/services.md) |
| [report/report.go](<../../report/report.go>) | report/report.go 탐색 그래프의 finding 노드를 가벼운 Markdown 보고서로 렌더링한다. | [가이드](modules/services.md) |

## screenshots — 15개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [screenshots/.gitkeep](<../../screenshots/.gitkeep>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/agents.png](<../../screenshots/agents.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/assets.png](<../../screenshots/assets.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/assets_test.png](<../../screenshots/assets_test.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/chat.png](<../../screenshots/chat.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/dashboard.png](<../../screenshots/dashboard.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/findings.png](<../../screenshots/findings.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/graph.png](<../../screenshots/graph.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/intercept.png](<../../screenshots/intercept.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/llm.png](<../../screenshots/llm.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/logs.png](<../../screenshots/logs.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/sessions.png](<../../screenshots/sessions.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/tasks.png](<../../screenshots/tasks.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/traffic.png](<../../screenshots/traffic.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |
| [screenshots/wx.png](<../../screenshots/wx.png>) | 원본 화면/커뮤니티 이미지 또는 자리표시자. README 미리보기와 원문 출처를 보존합니다. | [가이드](development.md) |

## selfupdate — 5개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [selfupdate/bootstrap.go](<../../selfupdate/bootstrap.go>) | selfupdate/bootstrap.go 서버 포트·DB 초기화 전에 준비된 업데이트를 확인하고 파일 교체 또는 복구를 결정한다. | [가이드](modules/services.md) |
| [selfupdate/github.go](<../../selfupdate/github.go>) | selfupdate/github.go GitHub의 최신 정식 릴리스와 플랫폼별 ZIP 자산을 찾는다. | [가이드](modules/services.md) |
| [selfupdate/selfupdate.go](<../../selfupdate/selfupdate.go>) | selfupdate/selfupdate.go 릴리스 바이너리를 다운받아 다음 시작에 교체하는 자가 업데이트의 공통 타입과 경로 규칙이다. | [가이드](modules/services.md) |
| [selfupdate/selfupdate_test.go](<../../selfupdate/selfupdate_test.go>) | selfupdate/selfupdate_test.go 임시 폴더와 짧은 가짜 실행 파일로 교체·실행 확인·롤백·체크섬·ZIP 추출을 검증한다. | [가이드](modules/services.md) |
| [selfupdate/stage.go](<../../selfupdate/stage.go>) | selfupdate/stage.go 업데이트의 다운로드 → ZIP 해시 대조 → 바이너리 추출 → -h 실행 확인 → .new 준비 단계를 담당한다. | [가이드](modules/services.md) |

## server — 102개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [server/assembly.go](<../../server/assembly.go>) | 역할별 도구·Skill·MCP 조립 wireAgentAugment는 DB의 역할별 가시성과 파일 시스템 Skill 정의를 읽어 세션에 추가할 도구 및 정리 함수를 만든다. | [가이드](modules/server.md) |
| [server/assembly_test.go](<../../server/assembly_test.go>) | assembly_test.go의 테스트 읽기 임시 Skill 디렉터리와 DB 가시성 설정을 만들고 wireAgentAugment 결과에 하나의 Skill 메타 도구가 실제로 포함되는지 확인한다. | [가이드](modules/server.md) |
| [server/asset_intercept.go](<../../server/asset_intercept.go>) | 전역 자산 차단 규칙의 HTTP 관리 domain·ip·url 및 CIDR 규칙을 생성·수정·삭제·활성화하는 API 계층이다. | [가이드](modules/server.md) |
| [server/assets.go](<../../server/assets.go>) | 기업·범위·공유 자산 API 기업 생성과 범위 추가는 입력 크기 및 검증 오류를 HTTP 상태로 변환하고 CompanyStore·AssetStore에 위임한다. | [가이드](modules/server.md) |
| [server/assets_scope_test.go](<../../server/assets_scope_test.go>) | assets_scope_test.go의 테스트 읽기 구형/구조화 범위 입력, 정규화된 중복 기업, 기업 삭제 뒤 살아 있는 Task의 연결 갱신, 본문 크기와 HTTP 400·404·500 분류를 검증한다. | [가이드](modules/server.md) |
| [server/auth.go](<../../server/auth.go>) | 단일 관리자 인증과 JWT 고정 사용자명 ARTEX, DB에 저장한 bcrypt 비밀번호 해시, 서명 키 파일을 이용하는 인증 구조다. | [가이드](modules/server.md) |
| [server/broadcast.go](<../../server/broadcast.go>) | 작업별 실시간 활동 배포 하나의 작업 ID에 여러 SSE 구독 채널을 연결하고 mutex로 등록·해제를 보호한다. | [가이드](modules/server.md) |
| [server/chat_llm_resolve_test.go](<../../server/chat_llm_resolve_test.go>) | chat_llm_resolve_test.go의 테스트 읽기 모델 미설정과 미활성 상태의 설명을 구분하고, 대화에 고정된 프로필이 실제 채팅 Agent 선택에 반영되는지 확인한다. | [가이드](modules/server.md) |
| [server/chat_mentions.go](<../../server/chat_mentions.go>) | 채팅의 구조화된 대상 참조 메시지 속 @[종류#숫자ID 표시명] 표기를 파싱해 중복을 제거하고 최대 개수·유효 ID를 확인한다. | [가이드](modules/server.md) |
| [server/chat_mentions_test.go](<../../server/chat_mentions_test.go>) | chat_mentions_test.go의 테스트 읽기 대상 참조 파싱·중복·페이지 처리·내용 상한을 검사하고, 모델 입력에 사용자가 쓴 표시명이 아닌 서버 DB 상세가 들어가는지 검증한다. | [가이드](modules/server.md) |
| [server/chatupload.go](<../../server/chatupload.go>) | 채팅 첨부 파일 저장과 모델 메시지 구성 작업 또는 대화의 작업 디렉터리에 첨부 파일을 저장하고 상대 경로·표시명·크기를 UI로 반환한다. | [가이드](modules/server.md) |
| [server/commands.go](<../../server/commands.go>) | 도구 이력·LLM 사용량 조회 API activity 기반 도구 실행 이력과 LLM 요청 기록·모델별 토큰·사용 통계를 페이지 및 작업 필터로 조회한다. | [가이드](modules/server.md) |
| [server/constraints_api.go](<../../server/constraints_api.go>) | 작업 운영 제약의 수동 편집 allow/deny 형태의 작업 제약을 task_constraints에 저장하며 Agent의 set_constraints 도구와 같은 데이터를 사용한다. | [가이드](modules/server.md) |
| [server/conversation_status_test.go](<../../server/conversation_status_test.go>) | conversation_status_test.go의 테스트 읽기 대화 목록의 running 표시가 서버의 실제 busy 상태와 일치하고 특정 Agent 종류에만 고정되지 않는지 확인한다. | [가이드](modules/server.md) |
| [server/conversations.go](<../../server/conversations.go>) | 독립 대화와 사용자 정의 Agent 트리거 큐 대화 CRUD, 메시지 저장, 실행 상태, 중단을 관리한다. 일반 대화는 대화별 하나의 실행만 허용하고 응답은 conversation_activities로 전달한다. | [가이드](modules/server.md) |
| [server/core_test.go](<../../server/core_test.go>) | core_test.go의 테스트 읽기 PostgreSQL-backed Manager와 HTTP 핸들러를 통해 작업 생성·자산/의도·활동·발견·보고서의 기본 경로가 이어지는지 검증한다. | [가이드](modules/server.md) |
| [server/customtool.go](<../../server/customtool.go>) | 사용자 정의 Python·명령·HTTP 도구 DB 도구 정의를 읽어 Norma CoreTool로 만들고 테스트 API와 Agent 실행에서 같은 실행 함수를 사용한다. | [가이드](modules/server.md) |
| [server/customtool_test.go](<../../server/customtool_test.go>) | customtool_test.go의 테스트 읽기 템플릿 치환·쉘 인용·JSON 스키마 기본값·Python stdin/환경 입력·로컬 HTTP 요청·큰 응답 잘림을 점검한다. Python 관련 테스트는 실제 로컬 프로세스를 사용한다. | [가이드](modules/server.md) |
| [server/dto.go](<../../server/dto.go>) | 저장 객체를 프런트엔드 계약으로 변환 Task·Node·Finding·Activity·Agent·LLMProfile을 HTTP JSON 형태로 바꾸는 전용 계층이다. | [가이드](modules/server.md) |
| [server/engine.go](<../../server/engine.go>) | 작업 실행 엔진과 Planner·Worker 협력 Run은 작업마다 Planner 루프 1개와 설정된 수의 Worker goroutine을 시작한다. goroutine은 OS 격리 경계가 아니다. | [가이드](modules/server.md) |
| [server/engine_cancelcause_test.go](<../../server/engine_cancelcause_test.go>) | engine_cancelcause_test.go의 테스트 읽기 작업 pause·개별 kill·사용자 의도 제어·삭제·수렴 종료의 취소 사유를 구분한다. 동시 제어 예약과 중단 후 상태 정착·재개 컨텍스트의 경쟁 조건도 고정한다. | [가이드](modules/server.md) |
| [server/engine_emptyturn_test.go](<../../server/engine_emptyturn_test.go>) | engine_emptyturn_test.go의 테스트 읽기 thinking만 있는 턴과 유효 응답을 구분하고, 도구/텍스트 없이 멈춘 턴을 한도 안에서 다시 유도하는 steerHooks.Stop 동작을 확인한다. | [가이드](modules/server.md) |
| [server/engine_llm_calls_test.go](<../../server/engine_llm_calls_test.go>) | engine_llm_calls_test.go의 테스트 읽기 여러 goroutine이 BeginLLMCall/EndLLMCall을 호출해도 작업별 실행 중 LLM 카운터가 일치하는지 검증한다. | [가이드](modules/server.md) |
| [server/engine_timeout.go](<../../server/engine_timeout.go>) | 작업 시간 예산과 순서 있는 종료 첫 실제 실행 시각을 기준으로 절대 deadline을 저장한다. 작업을 잠시 멈추어도 이미 시작된 벽시계 시간은 계속 흐른다. | [가이드](modules/server.md) |
| [server/finding_retests.go](<../../server/finding_retests.go>) | 발견 항목의 별도 재검증 작업 사용자가 발견 항목에 재검증을 요청하면 retester 대화를 만들고 원본 발견·증거를 연결한다. | [가이드](modules/server.md) |
| [server/finding_retests_test.go](<../../server/finding_retests_test.go>) | finding_retests_test.go의 테스트 읽기 재검증 시작·실행 중 목록·대화/도구 연결·중단·판정 없음·fixed 완료에 따른 원본 상태 갱신·범위를 검증한다. | [가이드](modules/server.md) |
| [server/finding_traffic.go](<../../server/finding_traffic.go>) | 발견에 묶인 HTTP 증거의 조회·편집 원시 트래픽 ID와 영속 증거 스냅샷을 구분하고 evidence.Store를 통해 바인딩·메모·순서·본문 조회를 처리한다. | [가이드](modules/server.md) |
| [server/finding_traffic_export.go](<../../server/finding_traffic_export.go>) | 발견과 증거를 ZIP 패키지로 묶기 발견 메타데이터, 증거 manifest, 요청·응답 파일을 하나의 ZIP에 기록하는 내보내기 계층이다. | [가이드](modules/server.md) |
| [server/finding_traffic_test.go](<../../server/finding_traffic_test.go>) | finding_traffic_test.go의 테스트 읽기 증거 바인딩·본문·ZIP 내보내기·아카이브 왕복·보고 실패 처리·UTF-8 구간·상속 자료의 쓰기 금지를 종합 검증한다. | [가이드](modules/server.md) |
| [server/finding_workflow.go](<../../server/finding_workflow.go>) | 발견 보고·증거 연결 도구의 서버 연결 발견 워크플로 도구를 DB에 등록하고 기존 사용자 설정을 보존하면서 필요한 스키마·기본 바인딩을 보완한다. | [가이드](modules/server.md) |
| [server/finding_workflow_test.go](<../../server/finding_workflow_test.go>) | finding_workflow_test.go의 테스트 읽기 발견에서 Planner 힌트로 이어지는 설정, 기존 사용자 도구 설정을 보존하는 마이그레이션, 보고서 작성 전 증거 바인딩 순서를 검증한다. | [가이드](modules/server.md) |
| [server/findings_groups.go](<../../server/findings_groups.go>) | 발견 그룹 탐색과 후속 심화 실행 HTTP 필터·페이지 크기를 정규화해 자산 트리 또는 작업별 발견 묶음을 반환한다. | [가이드](modules/server.md) |
| [server/findings_groups_test.go](<../../server/findings_groups_test.go>) | findings_groups_test.go의 테스트 읽기 잘못된 페이지 값의 정규화와 작업별 발견 묶음, 심화 요청의 감사 가능한 intent 생성·작업 재개·진입 실패 시 정리를 확인한다. | [가이드](modules/server.md) |
| [server/goals.go](<../../server/goals.go>) | 목표 초기화와 작업 실행 슬롯 배정 HTTP 작업 생성과 spawn_task 도구가 launchTask를 공유한다. 초기 자산·선택적 seed intent 이후 실행 슬롯을 배정한다. | [가이드](modules/server.md) |
| [server/goals_api.go](<../../server/goals_api.go>) | 목표 노드의 수동 관리 목표 목록·추가·수정·삭제를 제공하며 사용자의 변경을 탐색 그래프와 활동 기록에 남긴다. | [가이드](modules/server.md) |
| [server/goals_test.go](<../../server/goals_test.go>) | goals_test.go의 테스트 읽기 nil Task를 createGoals에 전달했을 때 모델 호출이나 패닉 없이 nil을 반환하는 작은 경계 테스트다. | [가이드](modules/server.md) |
| [server/inheritance_api_test.go](<../../server/inheritance_api_test.go>) | inheritance_api_test.go의 테스트 읽기 직접 연결한 원본 작업의 활동 상세를 읽는 경로와 상속 연결 삭제 후 접근 변화가 API에 반영되는지 확인한다. | [가이드](modules/server.md) |
| [server/inheritance_dto_test.go](<../../server/inheritance_dto_test.go>) | inheritance_dto_test.go의 테스트 읽기 상속 결과에 원본 출처를 붙이고, 원본의 살아 있는 intent와 관련 edge를 현재 작업 이력에 섞지 않는 필터를 검증한다. | [가이드](modules/server.md) |
| [server/intent_intervention.go](<../../server/intent_intervention.go>) | 사용자의 개별 Worker 후속 메시지 요청 ID를 검증하고 같은 의도에 중복 메시지 실행이 생기지 않도록 기록과 상태 전이를 확인한다. | [가이드](modules/server.md) |
| [server/intercept.go](<../../server/intercept.go>) | 도구 승인 규칙과 선택적 LLM 판정 API 도구별 승인 규칙, 대기 항목, 결정 이력, 상세 실행 내용을 HTTP로 노출한다. | [가이드](modules/server.md) |
| [server/intercept_detail_test.go](<../../server/intercept_detail_test.go>) | intercept_detail_test.go의 테스트 읽기 승인 항목의 상세 입력·출력·현재 실행 상태가 올바른 HTTP 계약으로 제공되는지 검증한다. | [가이드](modules/server.md) |
| [server/intercept_filter_test.go](<../../server/intercept_filter_test.go>) | intercept_filter_test.go의 테스트 읽기 승인 이력의 작업·동작·유형·페이지 필터와 잘못된 쿼리 처리를 HTTP 수준에서 점검한다. | [가이드](modules/server.md) |
| [server/intercept_live_test.go](<../../server/intercept_live_test.go>) | intercept_live_test.go의 테스트 읽기 실행 중 도구의 작업/의도 문맥이 승인 요청으로 전달되고 사람이 결정한 결과가 대기 중 실행을 해제하는지 확인한다. | [가이드](modules/server.md) |
| [server/intercept_review_test.go](<../../server/intercept_review_test.go>) | intercept_review_test.go의 테스트 읽기 LLM 판정 요청에는 현재 도구 호출 정보만 들어가며 불필요한 이전 대화 이력이 섞이지 않는지 가짜 provider로 검사한다. | [가이드](modules/server.md) |
| [server/llmpool.go](<../../server/llmpool.go>) | 여러 LLM 프로필의 공용 대체 경로 프로필로 provider를 만들고 활성 프로필부터 순서화한 Pool과 실패 건강 상태 레지스트리를 구성한다. | [가이드](modules/server.md) |
| [server/llmpool_test.go](<../../server/llmpool_test.go>) | llmpool_test.go의 테스트 읽기 활성 모델을 선두에 두고 rank·ID 순서를 따르는 상태 목록이 실제 Pool 체인의 우선순위와 일치하는지 검증한다. | [가이드](modules/server.md) |
| [server/llmrec_raw_test.go](<../../server/llmrec_raw_test.go>) | llmrec_raw_test.go의 테스트 읽기 모델 제공자 HTTP의 실제 wire 요청·응답 본문이 원시 기록에 저장되는지 확인해 재구성된 메시지와 원문 기록을 구분한다. | [가이드](modules/server.md) |
| [server/llmretry.go](<../../server/llmretry.go>) | 여러 재시도 계층의 설정 연결 전역 정책과 프로필별 필드 재정의를 agent.RetryConfig로 합쳐 provider·스트림·Worker 수준에 전달한다. | [가이드](modules/server.md) |
| [server/llmretry_test.go](<../../server/llmretry_test.go>) | llmretry_test.go의 테스트 읽기 전역 기본·프로필별 필드 재정의·명시적 비활성·범위 보정을 표 기반 사례로 검증한다. | [가이드](modules/server.md) |
| [server/logsink.go](<../../server/logsink.go>) | 표준 로그의 메모리·DB·실시간 전달 log 출력은 stderr를 유지하면서 3000개 크기의 메모리 링과 구독 채널로 복제된다. | [가이드](modules/server.md) |
| [server/manager.go](<../../server/manager.go>) | 영속 저장소와 실행 중 작업 핸들의 소유자 Manager는 PostgreSQL·공유 자산·트래픽·승인기를 연결하고 Task 핸들을 관리한다. Task는 탐색 저장소 및 동시 접근용 상태 스냅샷을 가진다. | [가이드](modules/server.md) |
| [server/manager_delete_files_test.go](<../../server/manager_delete_files_test.go>) | manager_delete_files_test.go의 테스트 읽기 작업 소유 파일/transcript만 선택하는 삭제 범위, 없는 파일의 멱등 처리, staging 되돌리기, DB 삭제 실패 시 파일·트래픽 복원을 검증한다. | [가이드](modules/server.md) |
| [server/manager_lifecycle_test.go](<../../server/manager_lifecycle_test.go>) | manager_lifecycle_test.go의 테스트 읽기 상태 스냅샷이 내부 슬라이스를 복사하고 여러 goroutine에서 서로 다른 시점의 필드가 섞이지 않게 읽히는지 검사한다. | [가이드](modules/server.md) |
| [server/mcpdiscover.go](<../../server/mcpdiscover.go>) | MCP 연결과 도구 목록 캐시 connectMCP가 stdio 또는 HTTP 계열의 전송 방식을 선택하고 공통 클라이언트 인터페이스로 노출한다. | [가이드](modules/server.md) |
| [server/mgmt_test.go](<../../server/mgmt_test.go>) | mgmt_test.go의 테스트 읽기 모델·Agent·프롬프트·가시성 등 관리 API가 인증 요청을 받아 DB의 설정을 저장하고 다시 조회하는 통합 흐름을 검사한다. | [가이드](modules/server.md) |
| [server/notifier.go](<../../server/notifier.go>) | 발견 알림의 실제 전송 엔진 PostgreSQL의 알림 delivery 큐에서 임대한 항목을 읽어 실시간 또는 묶음 메시지를 구성하고 채널 어댑터로 보낸다. | [가이드](modules/server.md) |
| [server/notify_api.go](<../../server/notify_api.go>) | 알림 채널·전송 이력 관리 API 채널 종류·설정 스키마·필터를 제공하고 생성·수정·테스트 전송·전송 이력·재시도를 처리한다. | [가이드](modules/server.md) |
| [server/notify_api_test.go](<../../server/notify_api_test.go>) | notify_api_test.go의 테스트 읽기 로컬 수신기를 사용해 실시간/묶음 알림, 비밀값 마스킹과 보존, 필터, 중단 채널, 재시도·속도 제한·임대 예산·딥링크를 검증한다. | [가이드](modules/server.md) |
| [server/orchestration.go](<../../server/orchestration.go>) | Agent가 사용하는 작업 간 조정 도구 list_tasks·spawn_task·pause_task·add_hint·trace 조회·보고서 갱신 등 서버 권한이 필요한 도구를 정의한다. | [가이드](modules/server.md) |
| [server/platform_tools.go](<../../server/platform_tools.go>) | 플랫폼 자체를 관리하는 Agent 도구 Skill 생성·파일 수정, 사용자 도구 등록·변경, MCP 등록·변경, 호스트별 자산 삭제를 CoreTool로 제공한다. | [가이드](modules/server.md) |
| [server/platform_tools_test.go](<../../server/platform_tools_test.go>) | platform_tools_test.go의 테스트 읽기 Agent 관리 도구로 Skill을 만들고 파일을 바꾸었을 때 실제 디렉터리와 내용이 계약대로 갱신되는지 확인한다. | [가이드](modules/server.md) |
| [server/prompt_vars_test.go](<../../server/prompt_vars_test.go>) | prompt_vars_test.go의 테스트 읽기 공통 프롬프트 변수를 합칠 때 같은 이름은 중복하지 않고 서로 다른 변수는 유지하는지 확인한다. | [가이드](modules/server.md) |
| [server/scheduler.go](<../../server/scheduler.go>) | 사용자 정의 Agent 자동 트리거 감시 일정 주기 및 새 발견·목표 달성·작업 생성·시간 초과·도구 결과를 감시해 StartTriggeredRun에 실행 요청을 넣는다. | [가이드](modules/server.md) |
| [server/server.go](<../../server/server.go>) | HTTP 라우팅과 런타임 연결의 중심 New가 인증 키, Engine, 역할별 LLM 라우터, 도구·프롬프트, 스케줄러, 알림, 기록, 재시작 복원을 연결한다. | [가이드](modules/server.md) |
| [server/server_mgmt.go](<../../server/server_mgmt.go>) | 역할·프롬프트·도구·Skill·MCP 관리 API 런타임 정책과 설정을 수정하는 관리 면이다. 역할별 프롬프트 버전, 가시성, 도구 스키마, 모델 프로필을 DB에 저장한다. | [가이드](modules/server.md) |
| [server/side_questions.go](<../../server/side_questions.go>) | 현재 대화와 분리된 /btw 보조 질문 부모 대화·작업·Worker의 체크포인트를 읽어 별도 질문의 입력 스냅샷으로 사용한다. | [가이드](modules/server.md) |
| [server/side_questions_test.go](<../../server/side_questions_test.go>) | side_questions_test.go의 테스트 읽기 보조 질문의 준비·취소·재연결·동시 제한·부모 격리·프로필 변경 검증·삭제/아카이브 전 기록 배출·재시작 체크포인트를 확인한다. | [가이드](modules/server.md) |
| [server/skill_upload_test.go](<../../server/skill_upload_test.go>) | skill_upload_test.go의 테스트 읽기 Skill 이름과 상대 경로, Zstd/중국어/GBK ZIP 파일명, 지원하지 않는 방식·암호화·경로 이탈 거부, frontmatter의 인용된 이름을 검증한다. | [가이드](modules/server.md) |
| [server/skill_usage.go](<../../server/skill_usage.go>) | Skill 호출 계량용 래퍼 원래 Skill 도구를 감싸 호출 입력에서 Skill 이름을 읽고 RunInfo의 작업·세션·역할에 사용량을 귀속한다. | [가이드](modules/server.md) |
| [server/skill_zip.go](<../../server/skill_zip.go>) | Skill ZIP 형식과 파일명 호환 ZIP 압축 방식의 추가 해제기를 등록하고 UTF-8·GBK 등 파일명을 일관된 문자열로 읽는다. | [가이드](modules/server.md) |
| [server/sync_scopesentry.go](<../../server/sync_scopesentry.go>) | ScopeSentry MCP의 자산 가져오기 설정된 ScopeSentry HTTP MCP에 연결해 데이터 소스·프로젝트·작업 목록과 자산 검색을 수행한다. | [가이드](modules/server.md) |
| [server/task_admission_test.go](<../../server/task_admission_test.go>) | task_admission_test.go의 테스트 읽기 FIFO 공정성, 실행 한도 축소, 모델 불가 작업의 슬롯 반환, 재개·시간 초과 시계, DB 경쟁, 도구 pause 및 실패 복원을 검사한다. | [가이드](modules/server.md) |
| [server/task_archive_package.go](<../../server/task_archive_package.go>) | 아카이브 파일 이동·압축·복구의 물리 계층 작업 파일과 transcript를 같은 파일 시스템의 staging 위치로 옮기고 journal을 남겨 중단된 이동을 재시작 시 복구한다. | [가이드](modules/server.md) |
| [server/task_archive_package_test.go](<../../server/task_archive_package_test.go>) | task_archive_package_test.go의 테스트 읽기 파일 패키지 왕복, 심볼릭 링크 제외, 중단된 이동/설치 journal 복구, 삭제 staging 재개, 압축 경로 이탈 거부를 검증한다. | [가이드](modules/server.md) |
| [server/task_archives.go](<../../server/task_archives.go>) | 작업 아카이브의 비동기 작업 관리자 API는 아카이브·복원·삭제 요청을 영속 작업으로 등록하고 백그라운드 worker가 한 항목씩 처리한다. | [가이드](modules/server.md) |
| [server/task_archives_test.go](<../../server/task_archives_test.go>) | task_archives_test.go의 테스트 읽기 아카이브 요청이 즉시 실행 결과가 아닌 큐 항목으로 생성되고 목록·요청 한도가 API 계약을 지키는지 검사한다. | [가이드](modules/server.md) |
| [server/task_assets.go](<../../server/task_assets.go>) | 작업과 공유 자산의 연결 관리 공유 자산을 특정 작업에 붙이거나 떼고 의도가 참조하는 자산을 조회한다. | [가이드](modules/server.md) |
| [server/task_categories.go](<../../server/task_categories.go>) | 작업 분류와 일괄 이동 분류 이름 생성·수정·삭제, 한 작업 또는 여러 작업의 분류 변경을 처리한다. | [가이드](modules/server.md) |
| [server/task_categories_test.go](<../../server/task_categories_test.go>) | task_categories_test.go의 테스트 읽기 일괄 분류 이동에서 대상 작업과 요청 ID 정규화·오류 처리가 라우트에 연결되어 있는지 확인한다. | [가이드](modules/server.md) |
| [server/task_control.go](<../../server/task_control.go>) | 작업·의도 제어의 공통 상태 전이 HTTP와 Agent 도구가 공유하는 pause/resume 및 의도 제어를 구현한다. | [가이드](modules/server.md) |
| [server/task_control_routes_test.go](<../../server/task_control_routes_test.go>) | task_control_routes_test.go의 테스트 읽기 Worker 의도 제어 URL이 올바른 핸들러에 연결되고 잘못된/없는 의도를 의미 있는 상태로 응답하는지 확인한다. | [가이드](modules/server.md) |
| [server/task_delete_barrier_test.go](<../../server/task_delete_barrier_test.go>) | task_delete_barrier_test.go의 테스트 읽기 삭제 중 새 작업 연산 금지, 기존 pause 보존, 실행 슬롯 유지, 대화 종료 대기, 업로드 차단, 커밋 뒤 정리 경고 처리를 검증한다. | [가이드](modules/server.md) |
| [server/task_intercept.go](<../../server/task_intercept.go>) | 작업별 도구 승인 규칙 관리 전역 규칙과 별도로 작업에 귀속된 승인 규칙의 목록·생성·수정·삭제·활성화를 제공한다. | [가이드](modules/server.md) |
| [server/task_llm.go](<../../server/task_llm.go>) | 작업별 LLM 선택·재시도·잔액 소진 전환 실행 시 우선순위는 역할의 명시적 바인딩 → 작업의 순서 있는 프로필 체인 → 전역 provider다. 소진된 명시적 체인은 조용히 전역으로 우회하지 않는다. | [가이드](modules/server.md) |
| [server/task_llm_test.go](<../../server/task_llm_test.go>) | task_llm_test.go의 테스트 읽기 가짜 provider로 출력 전/후 오류를 재현해 안전한 재시도와 잔액 소진 체인 이동, revision 경쟁, 명시적 체인 고갈, 프로필 삭제 뒤 복구를 검사한다. | [가이드](modules/server.md) |
| [server/task_metadata.go](<../../server/task_metadata.go>) | 목록용 작업 이름·고정 표시 변경 PATCH 입력 크기와 이름 길이를 검증하고 이름·pin 같은 목록 메타데이터를 Manager에 전달한다. | [가이드](modules/server.md) |
| [server/task_metadata_test.go](<../../server/task_metadata_test.go>) | task_metadata_test.go의 테스트 읽기 작업 이름·pin PATCH의 응답과 영속 변경, 대화 일괄 삭제의 사라진 ID별 결과를 검증한다. | [가이드](modules/server.md) |
| [server/task_resolution.go](<../../server/task_resolution.go>) | 현재 역할이 사용할 LLM의 설명 API Planner·Worker·Main 등 역할별로 모델이 어디서 결정되는지 프로필 ID·모델·출처·가용성·이유를 반환한다. | [가이드](modules/server.md) |
| [server/task_templates.go](<../../server/task_templates.go>) | 재사용 가능한 작업 생성 템플릿 작업 이름·설명·목표·분류·작업별 승인 규칙 묶음을 저장하고 관리한다. | [가이드](modules/server.md) |
| [server/task_templates_test.go](<../../server/task_templates_test.go>) | task_templates_test.go의 테스트 읽기 작업 템플릿의 생성·조회·부분 수정·삭제·검증과 대화 pin PATCH의 반환 계약을 검사한다. | [가이드](modules/server.md) |
| [server/testmain_test.go](<../../server/testmain_test.go>) | testmain_test.go의 테스트 읽기 server 테스트 패키지 전체에서 PostgreSQL advisory lock을 잡아 db/agent 테스트의 공유 정리와 경쟁을 줄인다. 접속 실패 때도 순수 테스트는 실행한다. | [가이드](modules/server.md) |
| [server/tool_usage.go](<../../server/tool_usage.go>) | 도구 호출 직전 사용량 기록 meteredTool은 CoreTool을 포함해 원래 스키마와 권한 정보를 유지하며 Call만 감싼다. | [가이드](modules/server.md) |
| [server/tool_usage_test.go](<../../server/tool_usage_test.go>) | tool_usage_test.go의 테스트 읽기 계량 래퍼가 작업·역할·도구를 기록하고 원 Call에 위임하며, 계량 DB 실패 때문에 실제 도구 호출을 막지 않는지 검증한다. | [가이드](modules/server.md) |
| [server/tools_wire_test.go](<../../server/tools_wire_test.go>) | tools_wire_test.go의 테스트 읽기 DB 도구 바인딩이 역할별 필터에 반영되고 설명·스키마 기본값 재정의가 실행 인스턴스까지 전달되는지 확인한다. | [가이드](modules/server.md) |
| [server/trigger_merge_test.go](<../../server/trigger_merge_test.go>) | trigger_merge_test.go의 테스트 읽기 같은 작업의 많은 트리거를 합쳐도 긴 목표 문맥은 한 번만 들어가고, 섞인 작업·빈 문맥·최대 길이가 올바르게 처리되는지 검증한다. | [가이드](modules/server.md) |
| [server/triggers.go](<../../server/triggers.go>) | 사용자 정의 Agent의 자동 실행 조건 CRUD interval 및 발견·목표·작업·도구 호출 이벤트에 반응할 트리거 설정을 저장한다. | [가이드](modules/server.md) |
| [server/update.go](<../../server/update.go>) | 버전 확인·업데이트 진행과 재시작 요청 릴리스 조회 결과와 오류를 캐시해 외부 API 요청을 줄이고, 한 번에 하나의 교체 작업만 진행하도록 updateHub가 동시성을 관리한다. | [가이드](modules/server.md) |
| [server/update_test.go](<../../server/update_test.go>) | update_test.go의 테스트 읽기 릴리스 조회 성공/오류의 서로 다른 캐시 시간, 강제 갱신, 만료, 호출자 취소로 캐시가 오염되지 않는 동작을 검증한다. | [가이드](modules/server.md) |
| [server/webui_embed.go](<../../server/webui_embed.go>) | 정적 웹 UI를 Go 바이너리에 포함 embedui 빌드 태그가 있을 때만 선택되는 구현이다. server/webui/dist에 미리 만든 Next 정적 내보내기 결과가 필요하다. | [가이드](modules/server.md) |
| [server/webui_stub.go](<../../server/webui_stub.go>) | 정적 UI를 포함하지 않는 개발용 구현 embedui 태그가 없으면 선택된다. 같은 webuiHandler 이름을 제공하지만 UI 요청에는 안내용 404를 반환한다. | [가이드](modules/server.md) |
| [server/worker_message_test.go](<../../server/worker_message_test.go>) | worker_message_test.go의 테스트 읽기 제거된 intervene 제어를 거부하고 실행 중인 의도가 없을 때 충돌을 반환하며, 메시지 요청 ID 형식을 검증한다. | [가이드](modules/server.md) |
| [server/workspace.go](<../../server/workspace.go>) | Agent 작업 파일 탐색·편집 API 작업 루트 안의 파일 목록·텍스트 읽기/쓰기·디렉터리 생성·삭제·다운로드·업로드를 제공한다. | [가이드](modules/server.md) |

## sidequestion — 10개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [sidequestion/CONTEXT_BUDGET.md](<../../sidequestion/CONTEXT_BUDGET.md>) | 보조 질문의 입력 예산·요약·현재 질문 보호에 관한 한국어 설명입니다. | [가이드](modules/agents.md) |
| [sidequestion/README.md](<../../sidequestion/README.md>) | 주 대화와 독립적인 /btw 보조 질문 기능의 한국어 개요입니다. | [가이드](modules/agents.md) |
| [sidequestion/VALIDATION.md](<../../sidequestion/VALIDATION.md>) | 보조 질문의 검증 방법·실행 조건·한계에 관한 한국어 문서입니다. | [가이드](modules/agents.md) |
| [sidequestion/capture.go](<../../sidequestion/capture.go>) | 주 Agent의 안정된 모델 요청 경계를 독립 질문에 쓸 불변 스냅샷으로 만든다. 부모 키는 conversation 또는 task/exploration/intent 조합으로 정하며 재사용 Worker 슬롯과 구분한다. | [가이드](modules/agents.md) |
| [sidequestion/context.go](<../../sidequestion/context.go>) | 독립 질문의 오래된 문답과 주 문맥 복사본을 예산 안으로 정리한다. Memory.Through는 배열 위치 대신 영속 ordinal을 사용하여 페이지·재시작 이후에도 요약 범위를 유지한다. | [가이드](modules/agents.md) |
| [sidequestion/context_test.go](<../../sidequestion/context_test.go>) | 독립 질문의 긴 문맥 예산과 재시작 가능한 요약 메모리를 검사한다. 20문답 경계, UTF-8 장문, tool_use/result 묶음, 요약 실패·취소의 메모리 불변성, 최대 12번 호출과 한 번만 허용되는 초과 복구를 검증한다. | [가이드](modules/agents.md) |
| [sidequestion/request.go](<../../sidequestion/request.go>) | 독립 질문의 데이터 형식과 간단한 요청 조립·입출력 예산 계산을 정의한다. Exchange는 질문/답/상태/부모/순번/사용량/준비 정보를 보관하는 단위이다. | [가이드](modules/agents.md) |
| [sidequestion/service.go](<../../sidequestion/service.go>) | 별도 질문 하나를 직접 Provider에 보내 답을 누적하는 가장 작은 실행 계층이다. Agent Session, 도구 실행기, 주 transcript 기록기, 작업 모델 장애 전환 루프를 소유하지 않는다. | [가이드](modules/agents.md) |
| [sidequestion/sidequestion_test.go](<../../sidequestion/sidequestion_test.go>) | 불변 체크포인트와 실제 선택 모델 신원, 도구 쌍, 예산 조립, 도구 실행 없는 서비스, 주/독립 질문 동시성 및 취소 분리를 검증한다. 가짜 Provider가 원하는 시점에 응답하도록 제어하여 모델 속도에 의존하지 않는다. | [가이드](modules/agents.md) |
| [sidequestion/validation-2026-09-10.json](<../../sidequestion/validation-2026-09-10.json>) | 원본 지원 파일입니다. 연결된 모듈 가이드에서 역할과 보존 이유를 설명합니다. | [가이드](modules/agents.md) |

## skills — 21개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [skills/api-recon/SKILL.md](<../../skills/api-recon/SKILL.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/api-recon/reference.md](<../../skills/api-recon/reference.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/build_perm_tree.py](<../../skills/api-recon/scripts/build_perm_tree.py>) | 프런트 권한·메뉴 구조의 로컬 stub 생성 다운로드한 JS의 userRouteAuth와 route_map.json을 읽어 코드·경로·부모 관계를 추정합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/extract_route_map.py](<../../skills/api-recon/scripts/extract_route_map.py>) | 다운로드된 번들에서 화면 라우트 사전 추출 각 JS 파일에서 KEY/name/link 패턴을 찾고 가장 많은 일치 항목을 가진 파일을 route_map.json으로 저장합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/harvest_static.py](<../../skills/api-recon/scripts/harvest_static.py>) | SPA 정적 JavaScript 자료 수집 HTML의 스크립트 URL에서 시작해 webpack/Vite 청크 정보를 확장하고 API 후보 경로와 화면 라우트를 파일로 저장합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/package-lock.json](<../../skills/api-recon/scripts/package-lock.json>) | API Recon 참고 스크립트의 npm 의존성 또는 잠금 정보. 원문을 보존하고 skills.md에서 해설합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/package.json](<../../skills/api-recon/scripts/package.json>) | API Recon 참고 스크립트의 npm 의존성 또는 잠금 정보. 원문을 보존하고 skills.md에서 해설합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/preload.js](<../../skills/api-recon/scripts/preload.js>) | 페이지 시작 시 삽입하는 관찰·모의 응답 스크립트 fetch/XHR와 라우터를 감싸 요청 URL·메서드·일부 본문/헤더를 메모리에 수집하고 설정한 stub 응답으로 UI를 렌더링합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/runtime_harvest.js](<../../skills/api-recon/scripts/runtime_harvest.js>) | 브라우저 실행 중 API 관찰 참고 도구 설정 파일로 Chromium을 실행하고 요청 가로채기·명시적 stub·라우트 이동으로 프런트가 만드는 API 요청을 기록합니다. | [가이드](skills.md) |
| [skills/api-recon/scripts/spider_mpa.py](<../../skills/api-recon/scripts/spider_mpa.py>) | 전통적인 여러 페이지 사이트의 HTML 탐색 SPA 번들 대신 HTML 링크·폼·인라인 스크립트에서 경로와 메서드·필드 이름을 추출하는 BFS 참고 구현입니다. | [가이드](skills.md) |
| [skills/playwright-cli/SKILL.md](<../../skills/playwright-cli/SKILL.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/element-attributes.md](<../../skills/playwright-cli/references/element-attributes.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/playwright-tests.md](<../../skills/playwright-cli/references/playwright-tests.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/request-mocking.md](<../../skills/playwright-cli/references/request-mocking.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/running-code.md](<../../skills/playwright-cli/references/running-code.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/session-management.md](<../../skills/playwright-cli/references/session-management.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/storage-state.md](<../../skills/playwright-cli/references/storage-state.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/test-generation.md](<../../skills/playwright-cli/references/test-generation.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/tracing.md](<../../skills/playwright-cli/references/tracing.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/playwright-cli/references/video-recording.md](<../../skills/playwright-cli/references/video-recording.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |
| [skills/scopesentry/SKILL.md](<../../skills/scopesentry/SKILL.md>) | 에이전트가 읽는 원본 절차 문서. 실행 입력은 보존하고 docs/ko/skills/의 대응 경로에 한국어 동반 문서를 제공합니다. | [가이드](skills.md) |

## traffic — 10개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [traffic/archive.go](<../../traffic/archive.go>) | traffic/archive.go 작업 아카이브에 넣을 원시 트래픽의 휴대 가능한 스냅샷을 만들고 복원한다. | [가이드](modules/services.md) |
| [traffic/archive_test.go](<../../traffic/archive_test.go>) | traffic/archive_test.go 임시 SQLite/폴더에서 트래픽 아카이브 왕복과 중단된 삭제 복구를 검증한다. | [가이드](modules/services.md) |
| [traffic/evidence.go](<../../traffic/evidence.go>) | traffic/evidence.go 수집된 트래픽에서 생략되지 않은 본문을 evidence 저장소로 전달하는 연결부다. | [가이드](modules/services.md) |
| [traffic/passthrough_test.go](<../../traffic/passthrough_test.go>) | traffic/passthrough_test.go MITM 오류 뒤 투명 통과 전환 조건을 검증한다. | [가이드](modules/services.md) |
| [traffic/proxy_test.go](<../../traffic/proxy_test.go>) | traffic/proxy_test.go 출구 프록시 주소 검증·설정 갱신·에이전트용 로컬 URL 변환을 검증한다. | [가이드](modules/services.md) |
| [traffic/reclaim_test.go](<../../traffic/reclaim_test.go>) | traffic/reclaim_test.go 트래픽 삭제가 논리적인 행 제거뿐 아니라 실제 디스크 공간 회수로 이어지는지 검증한다. | [가이드](modules/services.md) |
| [traffic/traffic.go](<../../traffic/traffic.go>) | traffic/traffic.go HTTP 트래픽 수집기의 생성·기록·검색·삭제·공간 회수를 한 파일에서 연결한다. | [가이드](modules/services.md) |
| [traffic/traffic_store_test.go](<../../traffic/traffic_store_test.go>) | traffic/traffic_store_test.go 신규 SQLite+blob 저장 형식의 기록·본문 검색·목록 필터·동시 쓰기 계약을 검증한다. | [가이드](modules/services.md) |
| [traffic/traffic_test.go](<../../traffic/traffic_test.go>) | traffic/traffic_test.go 호스트 헤더 복원과 부분/정확 호스트 삭제, SQL·폴더 롤백 및 공유 blob 정리를 검증한다. | [가이드](modules/services.md) |
| [traffic/upgrade_test.go](<../../traffic/upgrade_test.go>) | traffic/upgrade_test.go 옛 SQLite DSN/auto_vacuum 설정에서 현재 버전으로 넘어가도 데이터가 유지되는지 검사한다. | [가이드](modules/services.md) |

## web — 210개

| 원본 파일 | 한국어 읽기 안내 | 상세 |
| --- | --- | --- |
| [web/.gitignore](<../../web/.gitignore>) | 프런트엔드의 설치/빌드 산출물과 로컬 파일 제외 규칙입니다. | [가이드](modules/web-core.md) |
| [web/.husky/pre-commit](<../../web/.husky/pre-commit>) | 커밋 전에 CSS 프리셋 목록을 다시 생성하고 생성 결과를 스테이징한다. 이어서 lint-staged가 스테이징된 소스에 Biome 검사/자동 수정을 적용한다. | [가이드](modules/web-core.md) |
| [web/LICENSE](<../../web/LICENSE>) | 포함된 프런트엔드 MIT 고지. 저작권·라이선스 원문을 그대로 유지합니다. | [가이드](modules/web-core.md) |
| [web/biome.json](<../../web/biome.json>) | Biome 포맷·검사 규칙. 실제 JSON을 보존하고 web-core.md에서 설명합니다. | [가이드](modules/web-core.md) |
| [web/components.json](<../../web/components.json>) | shadcn 컴포넌트 생성·스타일·별칭 설정. 기존 UI를 설명하는 도구 설정입니다. | [가이드](modules/web-core.md) |
| [web/media/dashboard.png](<../../web/media/dashboard.png>) | 프런트엔드 참고 대시보드 이미지. 원본 리소스를 보존합니다. | [가이드](modules/web-core.md) |
| [web/next.config.mjs](<../../web/next.config.mjs>) | Next.js 빌드·개발 서버의 실행 방식 선택. NEXT_EXPORT=1이면 web/out에 정적 HTML·JS를 생성하고, NEXT_PUBLIC_MOCK=1이면 데모용 데이터 경로를 사용한다. | [가이드](modules/web-core.md) |
| [web/package-lock.json](<../../web/package-lock.json>) | npm 의존성 해석 결과와 무결성. 원문 그대로 보존합니다. | [가이드](modules/web-core.md) |
| [web/package.json](<../../web/package.json>) | npm 명령·의존성·프로젝트 메타데이터. web-core.md와 development.md에서 해설합니다. | [가이드](modules/web-core.md) |
| [web/postcss.config.mjs](<../../web/postcss.config.mjs>) | CSS 빌드 시 Tailwind CSS의 PostCSS 플러그인을 등록한다. UI의 유틸리티 클래스와 테마 변수를 최종 스타일시트로 변환하는 진입점이다. | [가이드](modules/web-core.md) |
| [web/public/logo.png](<../../web/public/logo.png>) | 공개 로고 이미지. 원본 리소스를 보존합니다. | [가이드](modules/web-core.md) |
| [web/public/logo.svg](<../../web/public/logo.svg>) | 벡터 로고. 경로 데이터와 원본 리소스를 보존합니다. | [가이드](modules/web-core.md) |
| [web/src/app/(auth)/layout.tsx](<../../web/src/app/(auth)/layout.tsx>) | 인증 화면의 공통 배치 로그인과 최초 비밀번호 설정 페이지의 children을 Fragment로 그대로 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(auth)/login/page.tsx](<../../web/src/app/(auth)/login/page.tsx>) | 로그인과 사용 조건 확인 고정 사용자명 ARTEX와 입력 비밀번호로 api.login을 호출하고 반환 토큰을 auth 모듈에 저장한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(auth)/setup/page.tsx](<../../web/src/app/(auth)/setup/page.tsx>) | 첫 실행의 관리자 비밀번호 설정 api.authStatus로 서버 초기화 여부를 확인하고 이미 설정된 서버는 로그인 화면으로 돌려보낸다. | [가이드](modules/web-pages.md) |
| [web/src/app/(external)/page.tsx](<../../web/src/app/(external)/page.tsx>) | 루트 주소의 진입 경로 웹사이트 루트 / 요청을 작업 목록 /function/tasks로 연결하는 작은 서버 페이지다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/main-content.tsx](<../../web/src/app/(main)/_components/main-content.tsx>) | 공통 상단 바와 본문 크기 조절 현재 경로를 보고 일반 화면에는 제목·탐색·사용자 메뉴·버전을 제공하고, 자체 헤더를 가진 화면에는 중복 여백을 넣지 않는다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/account-switcher.tsx](<../../web/src/app/(main)/_components/sidebar/account-switcher.tsx>) | 계정 표시와 비밀번호 메뉴 부모가 전달한 users 배열을 메뉴로 보여 주고 선택한 항목을 로컬 activeUser 상태로 표시한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/app-sidebar.tsx](<../../web/src/app/(main)/_components/sidebar/app-sidebar.tsx>) | 사이드바의 조립 지점 정의된 탐색 항목과 현재 사용자 정보를 NavMain·NavUser 등의 공통 조각에 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/change-password-dialog.tsx](<../../web/src/app/(main)/_components/sidebar/change-password-dialog.tsx>) | 관리자 비밀번호 변경 폼 현재 비밀번호·새 비밀번호·확인 입력을 별도로 보관하고 일치 여부를 검사한 뒤 api.changePassword로 제출한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/layout-controls.tsx](<../../web/src/app/(main)/_components/sidebar/layout-controls.tsx>) | 테마와 레이아웃 개인 설정 테마 모드·프리셋·본문 폭·상단 바·사이드바·글꼴을 usePreferencesStore의 getter와 setter에 연결한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/nav-documents.tsx](<../../web/src/app/(main)/_components/sidebar/nav-documents.tsx>) | 문서형 보조 탐색 메뉴 이름·주소·아이콘을 받은 항목을 사이드바 행과 부가 메뉴로 렌더링한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/nav-main.tsx](<../../web/src/app/(main)/_components/sidebar/nav-main.tsx>) | 활성 경로와 접힘 상태가 있는 주 탐색 현재 URL과 메뉴의 하위 항목을 비교해 선택 상태와 기본 펼침 상태를 계산한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/nav-secondary.tsx](<../../web/src/app/(main)/_components/sidebar/nav-secondary.tsx>) | 간단한 보조 링크 목록 부모가 전달한 항목을 아이콘과 제목이 있는 SidebarMenu로 매핑하는 표시 컴포넌트다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/nav-user.tsx](<../../web/src/app/(main)/_components/sidebar/nav-user.tsx>) | 사이드바 사용자 메뉴 전달받은 이름·이메일·아바타를 표시하고 비밀번호 변경 대화상자와 로그아웃 동작을 제공한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/search-dialog.tsx](<../../web/src/app/(main)/_components/sidebar/search-dialog.tsx>) | 화면 이동용 명령 검색 sidebarItems를 평탄화해 메뉴 이름과 그룹을 검색 대상으로 만들고 준비되지 않은 항목을 제외한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/sidebar-support-card.tsx](<../../web/src/app/(main)/_components/sidebar/sidebar-support-card.tsx>) | 사이드바의 지원 안내 카드 지원 안내 문구와 외부 링크를 정적으로 렌더링하는 작은 UI 조각이다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/sidebar/theme-switcher.tsx](<../../web/src/app/(main)/_components/sidebar/theme-switcher.tsx>) | 밝음·어두움·시스템 테마 순환 THEME_CYCLE 순서에 따라 현재 테마 모드를 다음 값으로 바꾸고 환경설정 저장소의 setter를 호출한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/_components/update-badge.tsx](<../../web/src/app/(main)/_components/update-badge.tsx>) | 상단 바의 새 버전 알림 마운트 시 api.checkUpdate를 한 번 호출하고 has_update가 참인 응답만 배지로 표시한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/chat/page.tsx](<../../web/src/app/(main)/chat/page.tsx>) | 작업 밖의 일반 에이전트 대화 화면 왼쪽에는 대화 목록·에이전트 그룹·고정과 이름 변경을, 오른쪽에는 초안 또는 선택 대화의 이력을 렌더링한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/dashboard/page.tsx](<../../web/src/app/(main)/dashboard/page.tsx>) | 여러 자원의 운영 현황 집계 작업·발견·활동·토큰·트래픽·자산·도구 설정을 여러 API에서 가져와 카드와 차트용 값으로 변환한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/assets/page.tsx](<../../web/src/app/(main)/function/assets/page.tsx>) | 전역 자산과 기업 범위 관리 기업 및 여섯 자산 종류를 탭으로 나누고 DSL·페이지 크기·페이지 번호를 서버 조회 조건으로 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/commands/page.tsx](<../../web/src/app/(main)/function/commands/page.tsx>) | 실행 도구 기록 검색 api.commands로 도구 호출 기록을 페이지 단위로 조회하고 api.commandStats로 집계 정보를 읽는다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/findings/_components/asset-tree.tsx](<../../web/src/app/(main)/function/findings/_components/asset-tree.tsx>) | 발견을 자산 계층으로 탐색하는 트리 평탄한 FindingAssetNode 배열을 부모·자식 트리로 조립하고 서버가 준 순서를 유지한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/findings/_components/findings-table.tsx](<../../web/src/app/(main)/function/findings/_components/findings-table.tsx>) | 발견 목록의 공통 표 본문 전역 평탄 목록과 작업별 그룹 목록이 동일한 선택·펼침·이름/분류 수정·상태 변경·재검증·삭제 행을 공유한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/findings/detail/lineage.tsx](<../../web/src/app/(main)/function/findings/detail/lineage.tsx>) | 특정 발견의 도출 경로 api.findingLineage로 해당 발견에 연결되는 탐색 노드·관계를 가져와 공통 ExplorationGraph에 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/findings/detail/page.tsx](<../../web/src/app/(main)/function/findings/detail/page.tsx>) | 발견 하나의 상세 검토 화면 URL 질의의 id로 api.getFinding을 조회하고 개요·도출 경로·트래픽 증거·재검증 기능을 조합한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/findings/page.tsx](<../../web/src/app/(main)/function/findings/page.tsx>) | 전역 발견 목록과 검토 작업 평탄 목록·작업별 그룹·자산 트리라는 세 보기 방식을 서로 다른 페이지 상태로 관리한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/llm-records/page.tsx](<../../web/src/app/(main)/function/llm-records/page.tsx>) | LLM 요청과 응답의 기록 조회 LLM 원문 기록 활성화 여부를 settings로 관리하고 세션·모델·작업 조건의 기록을 페이지 단위로 읽는다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/sync/page.tsx](<../../web/src/app/(main)/function/sync/page.tsx>) | ScopeSentry 자산 가져오기 연결 상태 카드가 데이터 소스 URL과 API 키를 설정하고, 작업 영역은 프로젝트 또는 작업 단위의 가져올 대상을 고른다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/assets-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/assets-tab.tsx>) | 작업에 연결된 자산 전역 자산 화면과 같은 DSL 입력을 사용하되 모든 조회에 taskId를 전달해 현재 작업의 자산과 출처를 보여 준다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/broadcast-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/broadcast-tab.tsx>) | 탐색 노드의 시간순 활동판 그래프 노드와 이웃 관계·앵커 자산을 explorationNodes 응답으로 받아 시간순 목록으로 보여 준다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/coverage-graph-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/coverage-graph-tab.tsx>) | 자산 범위와 연결 사실의 그래프 taskCoverageGraph 응답의 자산 계층을 G6 노드로 매핑하고 같은 부모 아래 많은 자식은 처음 20개 뒤에 접힘 노드를 붙인다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/findings-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/findings-tab.tsx>) | 한 작업의 발견 목록 api.findings(taskId)를 3초마다 읽고 저장된 정렬 선호에 따라 행을 배치한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/graph-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/graph-tab.tsx>) | 전체 탐색 그래프를 읽는 탭 api.explorationGraph를 20초마다 조회하고 데이터 서명이 달라졌을 때만 노드와 관계 상태를 갱신한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/intercept-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/intercept-tab.tsx>) | 작업 범위의 승인 기록 연결 공통 ApprovalRecords에 taskId를 전달하여 현재 작업의 도구 승인 기록만 보이도록 하는 얇은 어댑터다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/overview-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/overview-tab.tsx>) | 목표·제약·범위와 진행 상태의 총괄 작업·통계·의도·발견·커버리지·토큰을 함께 읽어 현재 진행 상태를 설명한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/report-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/report-tab.tsx>) | 작업 Markdown 보고서 보기 taskId가 바뀔 때 api.report의 텍스트를 읽고 공통 Markdown 렌더러로 표시한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/retests-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/retests-tab.tsx>) | 작업 발견의 재검증 모음 현재 작업의 발견을 20개씩 조회하고 각 발견에 공통 FindingRetestPanel을 붙인다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/_tabs/sessions-tab.tsx](<../../web/src/app/(main)/function/tasks/detail/_tabs/sessions-tab.tsx>) | 주 대화·Planner·Worker 실행 이력 main:<구간>·plan·intent:<번호>·system 키별로 별도의 역방향 페이지 캐시를 유지한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/detail/page.tsx](<../../web/src/app/(main)/function/tasks/detail/page.tsx>) | 작업 상세의 헤더와 열 개 탭 URL의 id로 작업과 통계를 5초마다 읽고 실시간 실행 상태를 영속 작업 정보에 보완한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/tasks/page.tsx](<../../web/src/app/(main)/function/tasks/page.tsx>) | 작업 생성·목록·분류·아카이브 작업 목록의 필터·정렬·고정·선택 상태를 관리하고 생성 폼에서는 목표·범위·모델 체인·상속할 작업·첨부·템플릿을 모은다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/traffic/page.tsx](<../../web/src/app/(main)/function/traffic/page.tsx>) | 수집된 HTTP 교환의 검색과 확인 호스트·본문·경로·상태·응답 크기·메서드·정렬 조건을 서버에 보내 트래픽을 페이지 단위로 조회한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/function/workspace/page.tsx](<../../web/src/app/(main)/function/workspace/page.tsx>) | 서버 작업 공간의 파일 탐색기 workspaceList 응답의 서버 기준 경로로 디렉터리와 이동 경로를 구성하고 파일 선택 시 workspaceRead를 호출한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/layout.tsx](<../../web/src/app/(main)/layout.tsx>) | 로그인 이후 화면의 공통 셸 토큰 존재 여부를 확인한 후 SidebarProvider·AppSidebar·MainContent를 구성한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/agents/detail/page.tsx](<../../web/src/app/(main)/system/agents/detail/page.tsx>) | 에이전트 편집기의 직접 링크 검색 매개변수로 전달된 에이전트 키를 읽고 공통 AgentEditor를 전체 너비로 표시한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/agents/page.tsx](<../../web/src/app/(main)/system/agents/page.tsx>) | 에이전트 목록과 사용자 에이전트 생성 api.agents로 역할별 카드 목록을 읽고 선택한 에이전트를 AgentEditor 서랍에서 편집한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/intercept/approvals/page.tsx](<../../web/src/app/(main)/system/intercept/approvals/page.tsx>) | 전역 승인 기록 페이지 taskId 없이 ApprovalRecords를 렌더링하여 작업 전체의 도구 승인 이력을 보여 준다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/intercept/assets/page.tsx](<../../web/src/app/(main)/system/intercept/assets/page.tsx>) | 전역 자산 허용·차단 규칙 편집 자산 일치 종류·패턴·동작·비고를 RuleForm으로 편집하고 전역 규칙 CRUD API에 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/intercept/page.tsx](<../../web/src/app/(main)/system/intercept/page.tsx>) | 도구 실행 규칙과 LLM 판단 설정 허용·거부·승인 요청 규칙, 적용할 도구 집합, 보조 LLM Judge 설정을 별도 API로 관리한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/llm/_components/retry.tsx](<../../web/src/app/(main)/system/llm/_components/retry.tsx>) | 다섯 재시도 계층의 공통 입력 접속·빈 응답·동일 provider 안전 창·풀 차단·의도 재실행의 횟수와 간격을 편집한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/llm/page.tsx](<../../web/src/app/(main)/system/llm/page.tsx>) | 모델 접속 프로필과 풀 상태 모델 형식·주소·키·출력 상한·thinking·reasoning_effort를 프로필로 저장하고 모델 조회/테스트를 제공한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/logs/page.tsx](<../../web/src/app/(main)/system/logs/page.tsx>) | 서버 로그의 실시간 조회 로그 SSE에서 새 줄을 받고 /api/logs/history로 오래된 기록을 추가 로드해 한 화면에 보여 준다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/mcp/page.tsx](<../../web/src/app/(main)/system/mcp/page.tsx>) | MCP 서버와 에이전트 가시성 설정 stdio·http·sse 전송 방식별 연결 정보를 저장하고 서버의 도구 목록을 조회·새로 고침한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/notify/_components/channel-fields.ts](<../../web/src/app/(main)/system/notify/_components/channel-fields.ts>) | 알림 채널의 입력 필드 사전 각 채널의 표시 이름·설명·필드 유형과 필터 선택지를 순수 데이터로 정의한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/notify/_components/channel-form.tsx](<../../web/src/app/(main)/system/notify/_components/channel-form.tsx>) | 알림 설정 입력과 필터 요약 필드 정의를 받아 텍스트·비밀번호·숫자·여러 줄 입력 중 알맞은 위젯을 만든다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/notify/_components/delivery-list.tsx](<../../web/src/app/(main)/system/notify/_components/delivery-list.tsx>) | 알림 전달 기록과 재전송 notifyDeliveries에 채널·상태·페이지 조건을 넘겨 전송 이력을 표시한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/notify/_components/stat-tile.tsx](<../../web/src/app/(main)/system/notify/_components/stat-tile.tsx>) | 알림 통계 숫자의 표시 조각 라벨·값·설명·색조를 받아 작은 통계 타일을 렌더링한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/notify/page.tsx](<../../web/src/app/(main)/system/notify/page.tsx>) | 알림 채널과 전송 정책 관리 채널 메타데이터와 현재 설정을 읽어 생성·수정·활성화·삭제·테스트 전송 요청을 조립한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/settings/_components/update-card.tsx](<../../web/src/app/(main)/system/settings/_components/update-card.tsx>) | 업데이트 확인과 재시작 추적 checkUpdate로 버전 정보를 읽고 사용자가 선택하면 applyUpdate 또는 rollbackUpdate를 호출한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/settings/page.tsx](<../../web/src/app/(main)/system/settings/page.tsx>) | 서버 기능 설정과 브라우저 선호 트래픽 수집·증거 자동 연결·검색·프록시·Python·Worker 수·제약 주입·실험 압축을 settings API에 연결한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/skills/page.tsx](<../../web/src/app/(main)/system/skills/page.tsx>) | Skill 파일과 가시성 편집기 Skill 목록을 파일 트리로 구성해 폴더·문서·스크립트의 읽기/쓰기/생성/삭제와 ZIP 업로드를 제공한다. | [가이드](modules/web-pages.md) |
| [web/src/app/(main)/system/tools/page.tsx](<../../web/src/app/(main)/system/tools/page.tsx>) | 기본·사용자 도구 목록과 편집 기본 도구의 JSON Schema를 행으로 풀어 설명과 기본값만 편집하고 핸들러에 고정된 이름·타입·필수 여부는 유지한다. | [가이드](modules/web-pages.md) |
| [web/src/app/favicon.ico](<../../web/src/app/favicon.ico>) | 브라우저 탭 아이콘. 이미지 바이트를 원본과 동일하게 보존합니다. | [가이드](modules/web-pages.md) |
| [web/src/app/globals.css](<../../web/src/app/globals.css>) | 전역 디자인 토큰과 기본 스타일 Tailwind와 애니메이션·shadcn 스타일을 가져온 뒤 CSS 변수들을 유틸리티 이름에 연결한다. | [가이드](modules/web-pages.md) |
| [web/src/app/icon.png](<../../web/src/app/icon.png>) | Next 앱 아이콘. 이미지 바이트를 원본과 동일하게 보존합니다. | [가이드](modules/web-pages.md) |
| [web/src/app/layout.tsx](<../../web/src/app/layout.tsx>) | 전체 문서의 루트 레이아웃 Next App Router의 가장 바깥 HTML과 공통 Provider를 조립한다. | [가이드](modules/web-pages.md) |
| [web/src/app/not-found.tsx](<../../web/src/app/not-found.tsx>) | 존재하지 않는 경로의 안내 화면 Next가 일치하는 페이지를 찾지 못했을 때 보여 주는 404 화면이다. | [가이드](modules/web-pages.md) |
| [web/src/components/agent-editor.tsx](<../../web/src/components/agent-editor.tsx>) | 에이전트 프롬프트·도구·트리거 편집기 에이전트 설정과 기본/종료/작업 시간 초과 프롬프트를 조회해 편집하고 미리보기 차이를 보여 준다. | [가이드](modules/web-pages.md) |
| [web/src/components/approval-execution-focus.tsx](<../../web/src/components/approval-execution-focus.tsx>) | 승인 기록에서 정확한 실행으로 이동 interceptExecution 응답으로 승인 대상 호출의 영속 위치를 찾고 현재 대화의 페이지 캐시를 그 위치까지 과거 방향으로 채운다. | [가이드](modules/web-pages.md) |
| [web/src/components/approval-records.tsx](<../../web/src/components/approval-records.tsx>) | 도구 승인 기록의 공통 검토 UI 전역 또는 taskId 범위의 이력 페이지와 아직 대기 중인 요청을 함께 읽고 5초마다 갱신한다. | [가이드](modules/web-pages.md) |
| [web/src/components/asset-dsl-search.tsx](<../../web/src/components/asset-dsl-search.tsx>) | 자산 검색 DSL 자동 완성 입력 필드·연산자·논리 연결자 목록과 커서 위치를 보고 다음에 입력할 후보를 계산한다. | [가이드](modules/web-pages.md) |
| [web/src/components/asset-intercept-rules-editor.tsx](<../../web/src/components/asset-intercept-rules-editor.tsx>) | 자산 규칙의 여러 행 입력 허용/차단 동작·자산 종류·일치값·비고를 props로 받은 배열에 대응시킨다. | [가이드](modules/web-pages.md) |
| [web/src/components/calendar/event-calendar-views.tsx](<../../web/src/components/calendar/event-calendar-views.tsx>) | FullCalendar의 여러 보기 형식을 ARTEX 테마에 맞추는 시각적 어댑터. 일/월/다중 월/시간표/목록에서 이벤트와 날짜 셀의 크기·선택·드래그·인쇄 상태별 클래스를 조합한다. | [가이드](modules/web-core.md) |
| [web/src/components/copy-button.tsx](<../../web/src/components/copy-button.tsx>) | 공통 클립보드 복사 버튼 copyText 유틸리티를 호출하여 HTTPS 클립보드와 가능한 대체 경로를 공유한다. | [가이드](modules/web-pages.md) |
| [web/src/components/date-range-picker.tsx](<../../web/src/components/date-range-picker.tsx>) | 날짜 범위 선택 위젯 Popover의 Calendar에서 선택한 시작/끝 날짜를 onChange로 부모에게 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/components/exploration-graph.tsx](<../../web/src/components/exploration-graph.tsx>) | 탐색 과정과 발견 계보의 공통 캔버스 노드 종류·상태·우선순위를 카드로 표현하고 관계를 따라 자동 배치한 React Flow 그래프를 그린다. | [가이드](modules/web-pages.md) |
| [web/src/components/finding-retest-dialog.tsx](<../../web/src/components/finding-retest-dialog.tsx>) | 발견 재검증 시작 대화상자 사용자의 추가 설명과 findingId로 startFindingRetest를 호출하고 생성된 실행을 onStarted에 돌려준다. | [가이드](modules/web-pages.md) |
| [web/src/components/finding-retest-panel.tsx](<../../web/src/components/finding-retest-panel.tsx>) | 발견별 재검증 이력 findingRetests로 실행 목록을 읽고 활성 실행이 있으면 3초 후 다시 조회한다. | [가이드](modules/web-pages.md) |
| [web/src/components/finding-traffic-panel.tsx](<../../web/src/components/finding-traffic-panel.tsx>) | 발견 증거 묶음의 순서와 역할 관리 findingTraffic에서 바인딩 목록과 version을 읽어 증거 순서·역할·설명을 편집하거나 연결을 제거한다. | [가이드](modules/web-pages.md) |
| [web/src/components/http-code-block.tsx](<../../web/src/components/http-code-block.tsx>) | HTTP 메시지의 읽기 쉬운 표시 첫 줄·헤더·본문을 분리하고 본문이 JSON·마크업·일반 텍스트인지 판단해 색상을 붙인다. | [가이드](modules/web-pages.md) |
| [web/src/components/link-traffic-dialog.tsx](<../../web/src/components/link-traffic-dialog.tsx>) | 선택 트래픽을 발견에 연결 트래픽 화면에서 고른 교환 목록을 받아 발견을 검색하고 역할/설명과 함께 bindFindingTraffic으로 전달한다. | [가이드](modules/web-pages.md) |
| [web/src/components/markdown.tsx](<../../web/src/components/markdown.tsx>) | Markdown의 공통 React 표시 react-markdown과 GFM 플러그인으로 제목·표·목록·코드·링크를 렌더링하고 각 요소에 Tailwind 클래스를 부여한다. | [가이드](modules/web-pages.md) |
| [web/src/components/mention-textarea.tsx](<../../web/src/components/mention-textarea.tsx>) | 자산·작업 등 참조 삽입 입력 커서 주변 입력을 분석해 참조 검색 조건을 만들고 chatMentions API에서 후보를 가져온다. | [가이드](modules/web-pages.md) |
| [web/src/components/scope-text-editor.tsx](<../../web/src/components/scope-text-editor.tsx>) | 범위 텍스트와 파싱 결과 표시 여러 줄 범위 입력을 부모의 onValueChange에 전달하고 parse 결과의 종류·규칙 수·오류를 시각화한다. | [가이드](modules/web-pages.md) |
| [web/src/components/side-question-workspace.tsx](<../../web/src/components/side-question-workspace.tsx>) | 보조 질문을 병렬로 보여 주는 작업 영역 SideQuestions 훅이 제공한 스레드·메시지·상태·조작 함수를 받아 보조 질문 버튼과 패널을 만든다. | [가이드](modules/web-pages.md) |
| [web/src/components/simple-icon.tsx](<../../web/src/components/simple-icon.tsx>) | Simple Icons의 SVG 래퍼 SimpleIcon 데이터의 경로·제목과 호출자가 전달한 SVG 속성을 실제 svg 요소로 옮긴다. | [가이드](modules/web-pages.md) |
| [web/src/components/status-badge.tsx](<../../web/src/components/status-badge.tsx>) | 도메인별 상태 표시 status 라이브러리에서 domain과 value에 해당하는 라벨·색조를 찾아 공통 배지로 렌더링한다. | [가이드](modules/web-pages.md) |
| [web/src/components/table-pagination.tsx](<../../web/src/components/table-pagination.tsx>) | 표의 페이지 이동 컨트롤 총 페이지 수와 현재 페이지를 받아 앞/뒤 및 페이지 번호 버튼을 구성한다. | [가이드](modules/web-pages.md) |
| [web/src/components/task-llm-profile-chain.tsx](<../../web/src/components/task-llm-profile-chain.tsx>) | 작업 LLM 순서 선택기 모델 프로필 ID의 순서 있는 배열을 받아 추가·제거·위치 이동과 현재 활성 프로필 선택을 제공한다. | [가이드](modules/web-pages.md) |
| [web/src/components/task-template-controls.tsx](<../../web/src/components/task-template-controls.tsx>) | 작업 생성 템플릿 선택과 관리 저장된 템플릿을 조회해 설명·목표·분류·자산 규칙을 생성 폼에 채우거나 현재 초안을 새 템플릿으로 저장한다. | [가이드](modules/web-pages.md) |
| [web/src/components/todo-popover.tsx](<../../web/src/components/todo-popover.tsx>) | 최근 TodoWrite 내용을 필요할 때 보기 호출자가 준 최신 TodoWrite seq와 detail 조회 함수를 사용해 열리는 시점에만 원문 JSON을 가져온다. | [가이드](modules/web-pages.md) |
| [web/src/components/traffic-evidence-viewer.tsx](<../../web/src/components/traffic-evidence-viewer.tsx>) | 저장된 HTTP 증거와 원본 교환 미리보기 TrafficEvidenceViewer는 발견에 바인딩된 스냅샷의 헤더·본문을 읽고 next_offset으로 텍스트 본문을 이어 붙인다. | [가이드](modules/web-pages.md) |
| [web/src/components/traffic-picker-dialog.tsx](<../../web/src/components/traffic-picker-dialog.tsx>) | 발견에 추가할 원본 트래픽 선택 발견 화면에서 호스트·메서드·문자열 조건으로 원본 트래픽을 조회하고 여러 교환을 선택한다. | [가이드](modules/web-pages.md) |
| [web/src/components/transcript.tsx](<../../web/src/components/transcript.tsx>) | 활동 기록을 대화와 실행 흐름으로 렌더링 Activity 배열을 사용자 입력·최종 답변·도구 쌍·설명·승인 카드로 묶어 재생한다. | [가이드](modules/web-pages.md) |
| [web/src/components/ui/accordion.tsx](<../../web/src/components/ui/accordion.tsx>) | 여러 항목을 접고 펴는 Radix Accordion 래퍼. Root/Item/Trigger/Content를 나누어 열림 상태와 키보드 동작은 Radix에 맡기고 제목 아이콘·전환 애니메이션을 입힌다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/alert-dialog.tsx](<../../web/src/components/ui/alert-dialog.tsx>) | 삭제 등 결정이 필요한 확인 대화상자의 구조와 스타일. Radix AlertDialog의 Root·Portal·Overlay·Content·Action·Cancel을 조합하고 크기별 레이아웃 및 설명/제목 접근성 연결을 제공한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/alert.tsx](<../../web/src/components/ui/alert.tsx>) | 인라인 안내/오류 메시지의 본문·제목·작업 영역. cva의 default/destructive 변형으로 표현 강도를 바꾸며 오류 판정과 메시지 데이터는 호출자가 전달한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/aspect-ratio.tsx](<../../web/src/components/ui/aspect-ratio.tsx>) | 컨텐츠의 가로세로 비율을 유지하는 Radix 컨테이너. ratio 등 원래 props를 그대로 전달한다. 이미지 로드나 리사이즈 작업 자체를 수행하는 컴포넌트는 아니다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/attachment.tsx](<../../web/src/components/ui/attachment.tsx>) | 대화 첨부파일 카드·미리보기·제목·설명·작업 버튼의 조립 부품. 크기와 방향 변형 및 상태 data-*로 표현을 바꾼다. 실제 업로드/다운로드는 호출자가 전달하는 동작으로 연결한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/avatar.tsx](<../../web/src/components/ui/avatar.tsx>) | 사용자 이미지·로드 실패 대체 표시·배지·아바타 그룹. Radix Avatar가 이미지 로딩 상태를 처리하고 이 파일은 크기별 모양과 그룹 겹침을 통일한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/badge.tsx](<../../web/src/components/ui/badge.tsx>) | 짧은 상태/개수 표시 배지와 시각 변형. cva가 variant별 클래스를 만들고 asChild이면 Radix Slot으로 자식 요소에 스타일과 props를 합친다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/breadcrumb.tsx](<../../web/src/components/ui/breadcrumb.tsx>) | 현재 페이지의 계층을 나타내는 경로 탐색 부품. nav/ol/li의 의미 구조, 현재 페이지 표시, 구분자, 생략 표시를 제공하며 경로 데이터는 호출자가 조립한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/bubble.tsx](<../../web/src/components/ui/bubble.tsx>) | 대화 메시지 말풍선의 묶음·본문·반응 영역. variant와 align에 따라 배경/정렬을 조정하며 메시지 저장이나 모델 응답 생성과는 독립적인 표시 계층이다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/button-group.tsx](<../../web/src/components/ui/button-group.tsx>) | 서로 붙어 있는 버튼 묶음의 방향·테두리·구분자 스타일. 가로/세로 방향에 따라 안쪽 모서리와 중복 테두리를 제거한다. 개별 버튼의 실행 함수는 그대로 전달된다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/button.tsx](<../../web/src/components/ui/button.tsx>) | 사이트 전반의 버튼 표현과 크기 규칙. cva로 variant/size를 타입에 연결하고 asChild이면 기본 button 대신 Slot을 사용한다. 클릭의 도메인 동작은 호출자가 정의한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/calendar.tsx](<../../web/src/components/ui/calendar.tsx>) | 날짜 선택용 react-day-picker를 공통 버튼·테마로 감싼 컴포넌트. 월 이동, 범위 선택, 바깥 날짜, 포커스된 날짜 버튼을 표현하며 locale/formatters/components 재정의를 전달한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/card.tsx](<../../web/src/components/ui/card.tsx>) | 카드의 헤더·제목·설명·작업·본문·푸터 구조. data-slot과 size를 이용해 카드 간 여백 및 자식 영역의 모양을 일관되게 만들며 데이터를 직접 조회하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/carousel.tsx](<../../web/src/components/ui/carousel.tsx>) | Embla 기반 가로/세로 캐러셀과 이전/다음 버튼. Context로 API·스크롤 함수·이동 가능 여부를 자식에 전달하고 select/reInit 이벤트에 따라 버튼 상태를 갱신한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/chart.tsx](<../../web/src/components/ui/chart.tsx>) | Recharts 차트를 테마·반응형 크기·범례·툴팁과 연결하는 계층. ChartConfig의 데이터 키별 이름/색을 Context로 공유하고 차트 고유 ID에 한정한 CSS 변수를 생성해 여러 차트의 색이 섞이지 않게 한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/checkbox.tsx](<../../web/src/components/ui/checkbox.tsx>) | 선택/미선택/중간 상태를 표현하는 Radix 체크박스. Indicator의 체크 또는 마이너스 아이콘과 aria-invalid/disabled 스타일을 제공하고 선택 값은 원래 props 계약으로 전달한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/collapsible.tsx](<../../web/src/components/ui/collapsible.tsx>) | 단일 영역을 접고 펼치는 Radix Collapsible 조립 부품. Root에 상태, Trigger에 토글, Content에 내용을 연결하며 여러 항목 간 선택 정책은 이 파일에서 추가하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/combobox.tsx](<../../web/src/components/ui/combobox.tsx>) | 검색 가능한 선택 및 여러 값의 칩 표시를 위한 Base UI Combobox 래퍼. 입력·트리거·초기화·팝업·목록·항목·칩을 분리하고 anchor를 기준으로 팝업을 배치한다. 후보 데이터와 검색 정책은 호출자가 제공한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/command.tsx](<../../web/src/components/ui/command.tsx>) | cmdk 명령 팔레트의 검색 입력·목록·항목·단축키 표시. Dialog 안에 넣는 CommandDialog도 제공한다. 여기서 말하는 명령은 UI 항목 선택이며 운영체제 명령 실행 함수가 아니다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/context-menu.tsx](<../../web/src/components/ui/context-menu.tsx>) | 마우스 보조 클릭으로 여는 Radix 컨텍스트 메뉴. 하위 메뉴·체크 항목·라디오 그룹·단축키 표기를 동일한 팝업 스타일로 감싸며 실행 콜백은 호출자가 준다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/dialog.tsx](<../../web/src/components/ui/dialog.tsx>) | 일반 모달 대화상자의 열기·닫기·Portal·본문·제목/설명 구조. Radix Dialog의 포커스/접근성 동작을 유지하며 Overlay와 Content에 공통 애니메이션과 크기 옵션을 적용한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/direction.tsx](<../../web/src/components/ui/direction.tsx>) | Radix 컴포넌트에 LTR/RTL 문서 방향을 공유한다. direction 속성이 있으면 dir보다 우선하며 useDirection을 그대로 노출한다. 문구 번역 기능과는 별개다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/drawer.tsx](<../../web/src/components/ui/drawer.tsx>) | Vaul 기반 드래그 가능한 서랍형 패널. 방향별 배치와 손잡이·제목·설명·푸터를 제공하고 열린 상태 및 제스처 처리는 Vaul에 맡긴다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/dropdown-menu.tsx](<../../web/src/components/ui/dropdown-menu.tsx>) | 버튼 트리거에서 여는 Radix 드롭다운 메뉴. Portal과 하위 메뉴, 선택/체크 항목을 조립하며 data-state에 따라 열림 애니메이션과 활성 스타일을 적용한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/empty.tsx](<../../web/src/components/ui/empty.tsx>) | 데이터가 없을 때 보여줄 아이콘·제목·설명·추가 작업 구조. 시각적인 빈 상태를 통일하며 비어 있음의 판정과 다음 행동은 화면에서 결정한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/field.tsx](<../../web/src/components/ui/field.tsx>) | 폼 필드의 묶음·레이블·설명·구분선·오류 표시. 수직/수평/반응형 배치를 지원한다. FieldError는 같은 message를 합쳐 한 오류 또는 목록으로 출력하지만 검증 규칙 자체는 실행하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/hover-card.tsx](<../../web/src/components/ui/hover-card.tsx>) | 요소 위에 포인터를 올렸을 때 부가 정보를 표시하는 Radix HoverCard. Trigger와 Portal 안의 Content를 제공하고 위치/간격 기본값과 열림 애니메이션을 설정한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/input-group.tsx](<../../web/src/components/ui/input-group.tsx>) | 아이콘·버튼·부가 텍스트를 입력/textarea와 하나의 테두리 안에 묶는다. Addon 클릭은 내부 버튼이 아닐 때 input에 포커스를 넘긴다. 그룹 안의 컨트롤은 자체 테두리를 지워 중복 표현을 피한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/input-otp.tsx](<../../web/src/components/ui/input-otp.tsx>) | 일회용 코드의 입력을 칸별로 표시하는 input-otp 어댑터. 실제 입력은 OTPInput이 관리하고 Slot은 Context의 문자·활성 여부·가상 커서를 읽어 렌더링한다. 코드 발급/검증 API는 포함하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/input.tsx](<../../web/src/components/ui/input.tsx>) | 한 줄 입력 요소의 기본 크기·포커스·오류·비활성 스타일. 네이티브 input의 props와 이벤트를 유지하고 클래스는 cn으로 호출자 설정과 병합한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/item.tsx](<../../web/src/components/ui/item.tsx>) | 목록 행의 미디어·제목·설명·추가 작업·머리/바닥 영역. variant와 size를 통해 카드형/일반 목록을 구성하며 항목 데이터와 링크/클릭 동작은 호출자가 전달한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/kbd.tsx](<../../web/src/components/ui/kbd.tsx>) | 키보드 키와 키 조합을 표시하는 작은 의미 요소. kbd 태그와 그룹 여백을 제공할 뿐 실제 키 이벤트를 등록하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/label.tsx](<../../web/src/components/ui/label.tsx>) | 입력 요소와 연결하는 Radix Label의 공통 스타일. htmlFor 등 원래 속성을 전달하여 레이블 클릭과 입력 포커스의 접근성 관계를 유지한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/marker.tsx](<../../web/src/components/ui/marker.tsx>) | 대화나 목록의 구분 표식·아이콘·본문 부품. 일반/양옆 구분선/아래 테두리 변형을 제공하며 asChild를 통해 링크 등 자식 요소와 합성할 수 있다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/menubar.tsx](<../../web/src/components/ui/menubar.tsx>) | 데스크톱 앱 형태의 수평 메뉴 바와 하위 메뉴. Radix Menubar의 키보드 이동·포커스·선택 상태를 유지하며 각 메뉴 항목을 공통 테마로 표시한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/message-scroller.tsx](<../../web/src/components/ui/message-scroller.tsx>) | 긴 대화의 스크롤 영역·메시지 항목·끝/처음 이동 버튼. @shadcn/react의 자동 스크롤 상태를 그대로 사용하며 content-visibility로 화면 밖 항목의 렌더 부담을 줄이는 스타일을 더한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/message.tsx](<../../web/src/components/ui/message.tsx>) | 대화 메시지의 묶음·아바타·본문·머리/바닥·작업 영역. 정렬과 간격 등 메시지 구조만 제공한다. 실제 활동의 스트림 연결·병합은 API/상위 transcript 계층의 책임이다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/native-select.tsx](<../../web/src/components/ui/native-select.tsx>) | 브라우저 기본 select/option/optgroup의 스타일 래퍼. 별도 Portal 팝업 없이 네이티브 선택 동작을 유지하고 크기와 화살표 위치만 맞춘다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/navigation-menu.tsx](<../../web/src/components/ui/navigation-menu.tsx>) | 사이트 상단 등의 탐색 메뉴와 드롭다운 viewport. Radix NavigationMenu에 트리거·콘텐츠·링크·표시선을 연결하며 viewport 사용 여부로 표시 구조를 바꿀 수 있다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/pagination.tsx](<../../web/src/components/ui/pagination.tsx>) | 페이지 번호·이전/다음·생략 부호의 탐색 구조. 활성 페이지는 aria-current와 버튼 변형으로 표현한다. 현재 페이지 상태와 서버 페이지 조회는 상위 화면이 처리한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/popover.tsx](<../../web/src/components/ui/popover.tsx>) | 클릭으로 여는 작은 부가 패널의 Root·Trigger·Content·Anchor. Portal로 표시하며 align과 sideOffset을 기본 설정한다. 모달보다 작은 편집/선택 UI를 조립하는 기반이다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/progress.tsx](<../../web/src/components/ui/progress.tsx>) | 0~100 값에 대응하는 수평 진행 막대. Indicator를 translateX로 이동시켜 남은 영역을 숨긴다. 실제 작업 진행률 계산·완료 판단은 외부에서 수행한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/radio-group.tsx](<../../web/src/components/ui/radio-group.tsx>) | 하나만 고르는 Radix 라디오 그룹과 항목. 그룹의 값/방향/키보드 계약을 유지하고 선택 Indicator·포커스·오류 스타일을 적용한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/resizable.tsx](<../../web/src/components/ui/resizable.tsx>) | 드래그로 크기를 조정하는 패널 그룹·패널·핸들. react-resizable-panels의 API를 전달하고 방향에 따른 flex 배치와 선택적 손잡이 모양을 입힌다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/scroll-area.tsx](<../../web/src/components/ui/scroll-area.tsx>) | 일정 영역 내부에 맞춤 스크롤바를 표시하는 Radix ScrollArea. Root/Viewport와 가로·세로 ScrollBar의 책임을 나누며 스크롤 내용과 크기는 호출자가 제공한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/select.tsx](<../../web/src/components/ui/select.tsx>) | Radix Select 기반 단일 값 선택 부품. Trigger에 현재 값을, Portal의 Content에 옵션/그룹/레이블을 표시하고 긴 목록의 위아래 이동 버튼을 제공한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/separator.tsx](<../../web/src/components/ui/separator.tsx>) | 레이아웃 영역 사이의 수평/수직 구분선. orientation은 모양과 의미를 결정하고 decorative 기본값은 보조 장식으로 취급하도록 Radix에 전달된다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/sheet.tsx](<../../web/src/components/ui/sheet.tsx>) | 화면 가장자리에서 열리는 Radix Dialog 기반 패널. side에 따라 상하좌우의 배치·애니메이션을 바꾸며 Portal·Overlay·닫기 버튼과 접근성 제목/설명을 함께 제공한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/sidebar.tsx](<../../web/src/components/ui/sidebar.tsx>) | 반응형 사이드바의 상태 저장소와 표시 부품 전체. 데스크톱 펼침과 모바일 열림 상태를 분리하고 Ctrl/Cmd+B 및 쿠키 저장을 연결한다. 화면 폭에 따라 고정 영역 또는 Sheet로 렌더링한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/skeleton.tsx](<../../web/src/components/ui/skeleton.tsx>) | 데이터 로딩 중 자리와 크기를 보여주는 펄스 애니메이션 요소. 로드 상태를 판단하거나 실제 콘텐츠를 조회하지 않고 호출자가 지정한 치수를 유지한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/slider.tsx](<../../web/src/components/ui/slider.tsx>) | 하나 또는 여러 손잡이로 수치를 고르는 Radix Slider. value/defaultValue 배열의 길이만큼 Thumb를 만들고 값이 없을 때 min/max를 표시 기준으로 사용한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/sonner.tsx](<../../web/src/components/ui/sonner.tsx>) | Sonner 토스트 알림의 테마·아이콘·색상 통일. next-themes에서 받은 모드를 전달하고 성공/정보/주의/오류/로딩 아이콘을 설정한다. 알림 발행은 toast를 호출하는 쪽에서 수행한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/sortable-head.tsx](<../../web/src/components/ui/sortable-head.tsx>) | 표의 정렬 가능한 열 제목과 현재 정렬 방향 표시. 클릭 시 field를 onSort에 전달하고 선택된 열만 방향 화살표를 표시한다. 정렬 실행 및 저장은 부모와 sort-preference 훅의 책임이다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/spinner.tsx](<../../web/src/components/ui/spinner.tsx>) | 작업 중임을 표시하는 회전 아이콘. status 역할과 스크린리더 레이블을 가진 SVG를 반환하며 시간 측정이나 작업 제어는 하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/switch.tsx](<../../web/src/components/ui/switch.tsx>) | 켜짐/꺼짐 설정을 위한 Radix Switch. 크기 변형과 checked/disabled/invalid 상태 스타일을 적용하며 저장 API 호출은 onCheckedChange를 받는 쪽에 있다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/table.tsx](<../../web/src/components/ui/table.tsx>) | 가로 스크롤 가능한 표와 thead/tbody/tfoot/tr/th/td/caption 부품. HTML 표 구조와 공통 표시를 유지한다. 행 데이터·선택·정렬·페이지네이션은 이 파일 밖에서 결합한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/tabs.tsx](<../../web/src/components/ui/tabs.tsx>) | 여러 패널을 전환하는 Radix Tabs의 목록·트리거·내용. 방향 및 탭 목록 변형을 스타일과 연결하고 value/onValueChange 계약을 그대로 전달한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/textarea.tsx](<../../web/src/components/ui/textarea.tsx>) | 여러 줄 입력의 자동 높이와 공통 폼 스타일. 내용에 따라 자라되 기본 최대 45vh 이후 스크롤하도록 하여 긴 붙여넣기가 다이얼로그 높이를 넘지 않게 한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/toggle-group.tsx](<../../web/src/components/ui/toggle-group.tsx>) | 같은 크기·모양·간격을 공유하는 토글 버튼 그룹. Context에 그룹 변형을 저장하고 Item은 그룹 설정 또는 개별 props에서 사용할 값을 정한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/toggle.tsx](<../../web/src/components/ui/toggle.tsx>) | 눌림/해제 상태가 있는 Radix 토글 버튼. cva의 크기·테두리 변형과 aria-pressed/data-state를 연결하며 일반 즉시 실행 Button과 구분한다. | [가이드](modules/web-core.md) |
| [web/src/components/ui/tooltip.tsx](<../../web/src/components/ui/tooltip.tsx>) | 포인터/포커스 시 짧은 설명을 보여주는 Radix Tooltip. Provider의 기본 지연은 0이며 Portal에 내용과 화살표를 렌더링한다. 보이지 않는 도움말 자체로 행동을 실행하지 않는다. | [가이드](modules/web-core.md) |
| [web/src/config/app-config.ts](<../../web/src/config/app-config.ts>) | 사이트 이름·메타 태그·저작권 표시의 공통 설정. 웹 패키지 버전은 package.json에서 읽으며, Go 바이너리의 실제 배포 버전은 api.health가 별도로 조회한다. | [가이드](modules/web-core.md) |
| [web/src/hooks/use-current-user.ts](<../../web/src/hooks/use-current-user.ts>) | 브라우저 마운트 후 JWT의 표시용 사용자 정보를 화면 상태에 반영하는 훅. 초기 렌더는 ARTEX 기본 사용자로 시작하며 auth.getCurrentUser가 성공하면 한 번 갱신한다. | [가이드](modules/web-core.md) |
| [web/src/hooks/use-lg.ts](<../../web/src/hooks/use-lg.ts>) | 화면 너비가 1024px 이상인지 구독하는 반응형 레이아웃 훅. 첫 렌더는 false이고 마운트 후 matchMedia의 change 이벤트와 innerWidth로 실제 값을 반영한다. | [가이드](modules/web-core.md) |
| [web/src/hooks/use-mobile.ts](<../../web/src/hooks/use-mobile.ts>) | 화면 너비가 768px 미만인지 구독하여 모바일 UI 분기를 결정한다. SSR/최초 렌더에서는 브라우저에 접근하지 않고, effect에서 너비를 읽은 뒤 change 이벤트를 구독한다. | [가이드](modules/web-core.md) |
| [web/src/hooks/use-side-questions.ts](<../../web/src/hooks/use-side-questions.ts>) | 주 대화에 붙은 /btw 보조질문 패널의 비동기 상태 관리. history 조회·2초 폴링·실행 중 질문의 SSE 스냅샷을 같은 목록으로 합친다. | [가이드](modules/web-core.md) |
| [web/src/lib/activity-merge.test.mjs](<../../web/src/lib/activity-merge.test.mjs>) | 활동 병합의 회귀 테스트: 겹치는 응답, 늦게 도착한 페이지, 토큰 중복, 입력 불변성을 검증한다. Node 내장 test/assert를 사용하며 네트워크·Go 서버·LLM 없이 순수 병합 함수만 실행한다. | [가이드](modules/web-core.md) |
| [web/src/lib/activity-merge.ts](<../../web/src/lib/activity-merge.ts>) | 히스토리 조회와 실시간 응답에서 겹친 활동을 seq 기준으로 병합한다. 기존 항목을 Map에 넣고 새 응답으로 같은 seq를 덮어쓴 다음 오름차순으로 반환한다. | [가이드](modules/web-core.md) |
| [web/src/lib/api.ts](<../../web/src/lib/api.ts>) | Go HTTP API와 웹 화면 사이의 공통 클라이언트 및 도메인별 요청 모음. http는 인증 헤더·오류 변환·204 응답을 처리하고, api의 각 메서드는 경로/JSON 필드와 응답 타입을 연결한다. | [가이드](modules/web-core.md) |
| [web/src/lib/auth.ts](<../../web/src/lib/auth.ts>) | 브라우저 인증 토큰 저장과 표시용 JWT 해석. localStorage는 API의 Bearer 헤더에, 동기화한 쿠키는 Next.js 페이지 이동 판단에 사용된다. | [가이드](modules/web-core.md) |
| [web/src/lib/chat-mentions.test.mjs](<../../web/src/lib/chat-mentions.test.mjs>) | @ 참조의 커서 처리·이메일 제외·중국어/영어 별칭·토큰 삭제 위치를 검사한다. 테스트의 중국어 문자열은 현재 직렬화 문법을 검증하는 입력/기대값이므로 번역 대상 설명과 구분한다. | [가이드](modules/web-core.md) |
| [web/src/lib/chat-mentions.ts](<../../web/src/lib/chat-mentions.ts>) | 대화 입력의 @ 참조 문법을 해석하고 자산/발견 선택 토큰으로 직렬화한다. activeMention은 커서 앞의 미완성 참조만 찾고 이메일 내부 @는 제외한다. | [가이드](modules/web-core.md) |
| [web/src/lib/chat-send-mode.ts](<../../web/src/lib/chat-send-mode.ts>) | Enter/Ctrl+Enter 전송 방식의 브라우저별 환경설정과 키 입력 판정. useSyncExternalStore로 같은 탭의 직접 알림과 다른 탭의 storage 이벤트를 함께 구독한다. | [가이드](modules/web-core.md) |
| [web/src/lib/company-scope.ts](<../../web/src/lib/company-scope.ts>) | 기업 범위 입력을 줄 단위로 분류·검사·정규화하는 순수 함수 모음. 도메인/URL, IPv4·IPv6, CIDR, ICP, 키워드를 구분하고 오류에는 원래 줄 번호를 남긴다. | [가이드](modules/web-core.md) |
| [web/src/lib/cookie.client.ts](<../../web/src/lib/cookie.client.ts>) | 브라우저 document.cookie에 대한 작은 읽기·쓰기·삭제 도우미. 환경설정 쿠키는 기본 7일 동안 path=/에 저장하고 삭제는 과거 만료일을 기록하는 방식이다. | [가이드](modules/web-core.md) |
| [web/src/lib/fonts/registry.ts](<../../web/src/lib/fonts/registry.ts>) | 글꼴 선택 키·표시 이름·Next.js 글꼴 변수의 연결표. 현재 원본 구현은 오프라인 빌드용으로 일반 글꼴 선택을 로컬 Geist Sans, 일부 고정폭 선택을 Geist Mono에 연결한다. | [가이드](modules/web-core.md) |
| [web/src/lib/local-storage.client.ts](<../../web/src/lib/local-storage.client.ts>) | 브라우저 localStorage 접근 실패를 흡수하는 환경설정 저장 도우미. 읽기가 막히면 null을 반환하고, 쓰기 실패는 개발 모드에서만 콘솔에 남긴다. | [가이드](modules/web-core.md) |
| [web/src/lib/mock/data.ts](<../../web/src/lib/mock/data.ts>) | 화면 데모용 정적 자료와 파생 데이터. 가상의 Acme 작업·자산·발견·탐색 그래프·활동·트래픽·설정을 같은 ID 관계로 연결해 화면을 채운다. | [가이드](modules/web-core.md) |
| [web/src/lib/mock/enabled.ts](<../../web/src/lib/mock/enabled.ts>) | NEXT_PUBLIC_MOCK=1인 빌드에서만 데모 데이터 경로를 여는 공통 플래그. 브라우저에 포함되는 환경 변수이며 실제 서버의 인증·LLM 실행 상태를 나타내는 값이 아니다. | [가이드](modules/web-core.md) |
| [web/src/lib/mock/handler.ts](<../../web/src/lib/mock/handler.ts>) | 데모 모드에서 HTTP 요청을 흉내 내는 메모리 라우터. method/path/query/body를 해석해 fixture를 조회하거나 복사본을 변경하며, 네트워크 대신 짧은 인공 지연을 준다. | [가이드](modules/web-core.md) |
| [web/src/lib/preferences/layout-utils.ts](<../../web/src/lib/preferences/layout-utils.ts>) | 레이아웃 선택을 document.documentElement의 data-* 속성으로 적용한다. 콘텐츠 폭·상단바·사이드바 형태·접기 방식·글꼴의 시각 효과는 이 속성을 읽는 CSS/컴포넌트가 담당한다. | [가이드](modules/web-core.md) |
| [web/src/lib/preferences/layout.ts](<../../web/src/lib/preferences/layout.ts>) | 사이드바 형태·접힘 방식·콘텐츠 폭·상단바 배치의 허용 선택값 정의. OPTIONS는 UI 표시용, VALUES는 검증용 목록, typeof로 파생한 타입은 코드의 값 범위를 제한한다. | [가이드](modules/web-core.md) |
| [web/src/lib/preferences/preferences-config.ts](<../../web/src/lib/preferences/preferences-config.ts>) | 환경설정 키마다 값 타입·기본값·저장 방식을 지정하는 중심 설정. 사이드바 형태와 접기 방식은 초기 레이아웃에 영향을 주므로 타입 수준에서 localStorage 선택을 금지한다. | [가이드](modules/web-core.md) |
| [web/src/lib/preferences/preferences-storage.ts](<../../web/src/lib/preferences/preferences-storage.ts>) | 선택한 환경설정의 지속 저장만 담당하는 어댑터. 설정표의 모드에 따라 무저장·쿠키·localStorage로 분기한다. | [가이드](modules/web-core.md) |
| [web/src/lib/preferences/theme-utils.ts](<../../web/src/lib/preferences/theme-utils.ts>) | 선택한 테마와 운영체제의 다크 모드를 실제 DOM 상태로 연결한다. system은 matchMedia로 light/dark를 결정하고 dark 클래스·colorScheme·data-theme-mode를 일관되게 갱신한다. | [가이드](modules/web-core.md) |
| [web/src/lib/preferences/theme.ts](<../../web/src/lib/preferences/theme.ts>) | 테마 모드와 프리셋 선택 목록의 타입/값 정의. 프리셋의 이름·값·밝은/어두운 대표 색은 styles/presets를 읽는 생성 스크립트가 표시된 구역에 써 넣는다. | [가이드](modules/web-core.md) |
| [web/src/lib/side-questions.ts](<../../web/src/lib/side-questions.ts>) | 보조질문 API 계약과 /btw 명령 인식. 질문은 부모 대화 경로에 연결되고 독립 id·진행 상태·답변 sequence·문맥 요약 정보를 가진다. | [가이드](modules/web-core.md) |
| [web/src/lib/sort-preference.ts](<../../web/src/lib/sort-preference.ts>) | 표의 정렬 열/방향을 브라우저에 보관하는 제네릭 훅. 저장값을 JSON으로 읽은 뒤 현재 허용 열 목록과 asc/desc를 확인한다. | [가이드](modules/web-core.md) |
| [web/src/lib/status.ts](<../../web/src/lib/status.ts>) | 작업·의도·발견·심각도·승인·알림 상태의 표시 이름과 색상 의미를 통일한다. 예를 들어 blocked는 실행 장애, exhausted는 예산 소진을 뜻하며 모두 단순한 완료/실패와 구분된다. | [가이드](modules/web-core.md) |
| [web/src/lib/task-assets.ts](<../../web/src/lib/task-assets.ts>) | 자산 종류와 작업 연결 출처를 화면 표시 문자열로 바꾼다. agent/anchor/manual/company 등 출처는 자산이 이 작업과 연결된 경로를 설명하며 자산 자체의 유형과 다르다. | [가이드](modules/web-core.md) |
| [web/src/lib/types.ts](<../../web/src/lib/types.ts>) | 프런트엔드가 기대하는 API 데이터 계약을 모은 타입 사전. Task는 실행 단위, Asset은 작업 사이에서 공유하는 자산, TaskNode/Edge는 탐색 과정, Finding은 별도 검토 상태를 갖는 발견이다. | [가이드](modules/web-core.md) |
| [web/src/lib/utils.ts](<../../web/src/lib/utils.ts>) | UI 공통 도우미: Tailwind 클래스 병합, 팝업 닫힘 조정, 복사, 이니셜, 통화 포맷. cn은 clsx의 조건부 클래스와 twMerge의 충돌 해결을 결합한다. | [가이드](modules/web-core.md) |
| [web/src/navigation/sidebar/sidebar-items.ts](<../../web/src/navigation/sidebar/sidebar-items.ts>) | 사이드바 메뉴의 계층·경로·아이콘을 한 곳에서 정의한다. 단일 링크와 하위 메뉴 부모를 타입으로 나누며 기능 화면과 시스템 설정 화면을 그룹화한다. | [가이드](modules/web-core.md) |
| [web/src/proxy.disabled.ts](<../../web/src/proxy.disabled.ts>) | 항상 요청을 통과시키는 Next.js Proxy 예제 파일. 이 파일명은 활성 진입점 proxy.ts와 다르며 현재 인증 리다이렉트 구현은 같은 폴더의 proxy.ts에 있다. | [가이드](modules/web-core.md) |
| [web/src/proxy.ts](<../../web/src/proxy.ts>) | Next.js 서버가 처리하는 페이지 요청의 로그인 화면 이동 규칙. 쿠키에 토큰이 있는지만 확인하여 로그인/초기화 화면과 작업 목록 사이를 이동시킨다. | [가이드](modules/web-core.md) |
| [web/src/scripts/generate-theme-presets.ts](<../../web/src/scripts/generate-theme-presets.ts>) | CSS 테마 메타데이터에서 TypeScript 프리셋 목록을 생성하는 개발 스크립트. 각 preset의 이름/키와 밝은·어두운 primary 색을 읽고 globals.css의 기본 테마를 앞에 추가한다. | [가이드](modules/web-core.md) |
| [web/src/scripts/theme-boot.tsx](<../../web/src/scripts/theme-boot.tsx>) | React hydration 전에 테마·글꼴·레이아웃을 적용하는 초기 스크립트 생성 컴포넌트. 환경설정 저장 방식과 기본값을 JSON으로 직렬화하여 head에서 실행할 코드 문자열에 넣는다. | [가이드](modules/web-core.md) |
| [web/src/stores/preferences/preferences-provider.tsx](<../../web/src/stores/preferences/preferences-provider.tsx>) | 컴포넌트 트리별 Zustand 저장소를 만들고 초기 DOM 테마와 동기화한다. ThemeBootScript가 이미 설정한 data-* 값을 허용 목록으로 읽어 store에 반영한 뒤 isSynced를 켠다. | [가이드](modules/web-core.md) |
| [web/src/stores/preferences/preferences-store.ts](<../../web/src/stores/preferences/preferences-store.ts>) | 테마·레이아웃·글꼴 선택의 메모리 상태와 setter를 정의하는 Zustand 저장소 팩터리. 전역 singleton 대신 createPreferencesStore를 호출해 Provider마다 초기값을 가진 저장소를 만든다. | [가이드](modules/web-core.md) |
| [web/src/styles/flag-icons/flags.css](<../../web/src/styles/flag-icons/flags.css>) | 국가/지역 코드에 대응하는 깃발 표시 스타일. flag:코드 형태의 클래스에 인코딩된 SVG data URL을 배경 이미지로 연결한다. | [가이드](modules/web-core.md) |
| [web/src/styles/presets/brutalist.css](<../../web/src/styles/presets/brutalist.css>) | 강한 대비와 선명한 강조색의 Brutalist 테마 토큰 정의. 같은 프리셋 키 아래 밝은 모드와 .dark 모드를 나누어 배경·텍스트·테두리·차트·사이드바 색 및 그림자를 지정한다. | [가이드](modules/web-core.md) |
| [web/src/styles/presets/soft-pop.css](<../../web/src/styles/presets/soft-pop.css>) | 둥근 모양과 부드러운 강조색의 Soft Pop 테마 토큰 정의. 같은 프리셋 키 아래 밝은 모드와 .dark 모드를 나누어 배경·텍스트·테두리·차트·사이드바 색 및 그림자를 지정한다. | [가이드](modules/web-core.md) |
| [web/src/styles/presets/tangerine.css](<../../web/src/styles/presets/tangerine.css>) | 주황 계열 강조색의 Tangerine 테마 토큰 정의. 같은 프리셋 키 아래 밝은 모드와 .dark 모드를 나누어 배경·텍스트·테두리·차트·사이드바 색 및 그림자를 지정한다. | [가이드](modules/web-core.md) |
| [web/tsconfig.json](<../../web/tsconfig.json>) | 웹 TypeScript 컴파일 옵션·경로 별칭·포함 범위. web-core.md에서 해설합니다. | [가이드](modules/web-core.md) |
| [web/tsconfig.scripts.json](<../../web/tsconfig.scripts.json>) | 테마 생성 스크립트용 TypeScript 설정. 앱 빌드 설정과 분리됩니다. | [가이드](modules/web-core.md) |
