# `/btw` 상위 프로젝트 검증 기록 — 한국어판

**기록 날짜:** 2026-09-10. **당시 브랜치:** `codex/btw-side-question`. **기준 커밋:** `8dae851b9b622f2ff2631f332fde9719d0b16fba`.

이 문서는 [원본 검증 기록](../docs/upstream/sidequestion-VALIDATION.md)의 번역입니다. 아래 통과·실패·실제 모델 호출은 **상위 프로젝트 작성자의 당시 기록**입니다. 한국어 주석 추가 과정에서 이 검증을 재실행하거나 외부 서비스에 요청했다는 뜻이 아닙니다.

당시 독립 PostgreSQL 테스트 DB와 데이터 디렉터리를 사용했고, 모델 자격증명은 별도 테스트 환경에만 주입하여 소스나 기록에 넣지 않았으며 제품 기본 모델도 바꾸지 않았다고 명시합니다. 당시 버전은 Go 1.26.3, Norma v0.3.6, Next.js 16.2.9였습니다. 현재 의존성은 [go.mod](../go.mod) 및 [web/package.json](../web/package.json)을 기준으로 합니다.

실제 대화, 응답 객체, 엔지니어링 단언, Qwen 원문 검토는 [validation-2026-09-10.json](validation-2026-09-10.json)에 원형 그대로 남아 있습니다. JSON의 `availability`는 모델 가용성, `streaming`은 스트리밍 시나리오, `restart`는 재시작, `non_streaming`은 완성 응답, `stream_usage_probe`는 사용량 프레임 확인, `qwen_review`는 보조 모델 검토입니다. 원본은 이 파일에 API 자격증명이 없다고 기록했습니다.

## 1. 엔지니어링 검사

| 범위 | 원본 결과 | 근거 테스트/명령 |
| --- | --- | --- |
| 구조화 메시지·도구 인자 깊은 복사 | 통과 | `TestCheckpointDeepCopyAndBoundaries` |
| 요약 호출의 덮어쓰기 방지, 완성 응답·종료 발행, 부분 응답 제외 | 통과 | `TestCheckpointDeepCopyAndBoundaries`, `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 실제 선택 모델 신원 | 통과 | `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 도구 쌍, 20문답 재생, 예산 축소·초과 오류 | 통과 | `TestBuildRequestCompactionToolPairingAndBudget` |
| 주 실행과 병렬, 양방향 취소 분리 | 통과 | 차단형 Provider, `TestMainSideConcurrencyAndIndependentCancellation` |
| 도구 미실행, 스트리밍/비스트리밍, 실패 전 사용량 | 통과 | `TestServiceNoToolsAndUsageOnFailure` |
| 실제 Norma ChatAgent·로컬 Read, transcript/활동 분리 | 통과 | `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`의 두 전송 방식 |
| 영속 저장·페이지·멱등·재시작 부분 답변 | 통과 | `TestSideHistoryIdempotencyPagingAndRecovery` |
| 삭제와 늦은 쓰기 경쟁, 부모 삭제, 버전 비교 | 통과 | `TestSideClearLateWritersAndDeletedParent` |
| MainAgent/Worker 아카이브 v1/v2/v3 | 통과 | `TestSideTaskArchiveVersions` |
| 세 부모 HTTP 경로·인증·소유권·Worker 논리 삭제 | 통과 | `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`, `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 바쁜 주 대화의 독립 질문·SSE 재연결·취소·삭제 | 통과 | `TestSideHTTPBusyIsolationClearAndReconnect` |
| 부모당 1개·전체 4개 동시 실행 | 통과 | 두 `TestSideHTTP…` 테스트 |
| 질문 전 스냅샷 저장, 재시작 후속 질문, 옛 세션 스냅샷 위조 금지 | 통과 | `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 캐시된 모델 설정 삭제·신원 변경 후 거부 | 통과 | `TestSideRejectsDeletedOrChangedCachedProfile` |
| 아카이브 전 취소 및 최종 답/사용량 저장 대기 | 통과 | `TestSideTaskDrainPersistsBeforeArchive` |
| 스트림 조기 중단의 사용량 1회 계측·독립 질문 귀속 | 통과 | `TestSideUsageRecordedOnceOnConsumerCancellation` |
| 재시작 복구 Worker·deadline 문맥의 새 체크포인트 | 통과 | `TestSideRestoredWorkerRuntimePublishesNewCheckpoint` |
| 관련 패키지 race 검사 | 통과 | 아래 명령 |
| TypeScript 및 프로덕션 빌드 | 통과 | `npx tsc --noEmit`, `npm run build` |
| 신규 프런트 모듈 Biome | 통과 | 신규 3모듈 `biome check` |

폐기 가능한 독립 DB를 `ARTEX_PG_DSN`에 설정한 뒤 당시 검사를 재현할 수 있습니다. 운영 DB를 지정하지 않습니다.

```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```

**전체 Go 회귀는 모두 통과한 것이 아닙니다.** `server`의 기존 두 테스트가 임시 폴더 정리에서 `TempDir RemoveAll … directory not empty`로 실패했습니다.

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

원본은 수정 전 기준 커밋을 같은 독립 환경에서 다시 실행해 두 정리 실패가 재현되었다고 기록했습니다. 기준 실행에는 `TestCoreTaskLifecyclePG`의 goal 개수 단언 실패도 추가로 있었으나 최종 변경본 회귀에서는 이 단언 실패가 나타나지 않았습니다. 다른 패키지, 독립 질문 관련 검사, race 검사는 통과했다고 구분합니다. 기존 실패를 이번 기능의 통과로 표시하거나 숨기려고 단언을 바꾸지 않았다는 기록입니다.

Next.js 빌드에는 기존 여러 lockfile 및 workspace root 추정 경고가 있었지만 빌드와 모든 페이지 생성은 완료되었습니다.

## 2. 브라우저 검사

당시 Codex In-app Browser를 독립 로컬 Go 서비스와 Next.js 개발 서버에 연결하고 데스크톱 및 390×844 화면에서 다음을 확인했습니다.

- 주 대화 도중 `/btw`를 제출해 주 내용과 독립 답이 동시에 표시되고 데스크톱 옆 패널이 동작했습니다.
- 후속 질문과 독립 질문 중지 이후 부분 답 보존을 확인했으며 주 실행은 계속되었습니다.
- 패널을 닫아도 요청이 이어지고 다시 열면 완료 답을 복구했으며 새로고침 후 빈 `/btw`로 이력을 열었습니다.
- 좁은 화면 Drawer의 입력·버튼·이력·닫기와 가로 넘침 없음을 확인했습니다.
- 삭제 확인창 뒤 독립 이력이 없어지되 주 transcript와 스냅샷은 유지되었습니다.
- MainAgent 및 Worker 두 개를 전환해 역할 라벨과 이력이 섞이지 않는지 확인했습니다.
- 차단형 로컬 모델 fixture로 Worker를 계속 실행시킨 상태에서 `/btw`를 제출했습니다. 독립 질문을 중지해도 Worker는 실행 중 표시와 자기 일시 정지 버튼을 유지했고 독립 부분 답을 저장했습니다.
- 브라우저 오류/경고 로그가 비어 있었습니다.

동시성 시점은 실제 모델의 속도가 아니라 제어 가능한 fixture로 검사했습니다. 처음 두 번의 Worker 점검은 작업 또는 답변이 일찍 끝나 유효한 동시 실행 구간을 만들지 못했습니다. fixture를 고쳐 다시 확인한 성공만 통과로 기록했습니다.

## 3. 실제 모델 대화 기록

당시 `grok-4.6`을 OpenAI 호환 `http://127.0.0.1:12580/tingly/openai`에서 우선 점검했습니다. HTTP 200, 응답 모델 `grok-4.6`, 텍스트 `READY`, 2.82초를 기록했습니다. 우선 모델이 사용 가능하여 Tingly `glm`과 Zhipu `glm-5.3` 대체 체인은 사용·검증하지 않았습니다.

| 시나리오 | 원본 기록 결과 |
| --- | --- |
| 주 대화 중 자산·목표·표식 질문 | `redhaze.top`, 홈페이지 읽기·정리 목표, `BTW-REAL-0910` 응답. 16.97초 |
| 주 대화가 홈페이지를 읽은 뒤 도구 근거 질문 | WebFetch 200, curl 301→302→200, 페이지 제목을 인용. 7.24초 |
| 독립 질문에서 Bash로 파일 생성 요구 | 실행을 거부했고 대상 파일이 생성되지 않음. 7.74초 |
| 독립 답변 뒤 주 문맥 불변성 | 주 transcript SHA-256 및 주 활동 동일, 독립 도구 실행 횟수 0 |
| Go 서비스를 실제 중지·재시작한 후 후속 질문 | 앞선 이력 3개와 영속 스냅샷으로 답변, 주 Agent를 다시 실행하지 않음 |
| 새 대화의 Grok 비스트리밍 | 자산 및 `ATOMIC-0910` 응답. input 11734, output 138, cache_read 11520을 저장 |

주 대화는 WebFetch와 Bash/curl로 공개 홈페이지를 읽었고 최종 URL은 `https://id.redhaze.top/home`, 제목은 원문 `红幕科技 RedHaze Group · 全球综合集团门户`였습니다. Bash는 응답을 로컬 임시 파일에 저장했고 원격 쓰기는 하지 않았다는 기록입니다. **주 대화의 로컬 파일 쓰기**와 **독립 질문의 도구 실행 0회**는 서로 다른 검사입니다.

기록된 주 transcript SHA-256은 `e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`입니다.

**사용량 한계:** 당시 Tingly의 Grok 스트림에는 usage가 없었습니다. `stream_options.include_usage=true`로 별도 확인해도 HTTP 200, 데이터 프레임 12개, usage 프레임 0개였습니다. 스트리밍 테스트의 사용량 0은 **끝점이 값을 제공하지 않음**을 뜻하며 무료 또는 무과금의 증거가 아닙니다. 비스트리밍과 fixture의 실패/취소 사용량은 저장되었습니다.

## 4. Qwen 보조 검토

원본은 `qwen-flash`를 OpenAI 호환 `https://dashscope.aliyuncs.com/compatible-mode/v1`에 호출해 HTTP 200을 받았다고 기록합니다. 초기 실제 독립 대화 세 건, 주 대화의 도구 근거, 엔지니어링 단언을 제공했고 `verdict: accept`, `concerns: []`를 반환했습니다. 사용량은 prompt 6625, completion 312, total 6937이었습니다.

이 검토에는 이후 추가한 서비스 재시작과 비스트리밍 검사가 포함되지 않았습니다. Qwen의 “쓰기 없음” 요약은 너무 넓었습니다. 주 대화 curl은 로컬 응답 파일을 만들었으므로 원본이 이를 명시적으로 정정했습니다. 동시성, 도구 실행 0회, transcript 분리는 엔지니어링 단언으로 판단하며 모델 검토는 답변 품질의 보조 자료입니다.
