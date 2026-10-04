# 원본 출처와 한국어 해설판 이력

이 저장소는 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)의 소스 스냅샷을
[Chang-Daegyu/ARTEX](https://github.com/Chang-Daegyu/ARTEX)에 복사한 독립 저장소입니다.
GitHub의 Fork 기능으로 생성하지 않았습니다. 원본 코드의 저자와 저작권은 그대로 존중합니다.

## 기준 소스

- 원본 저장소: `https://github.com/Autumn-27/ARTEX`
- 기준 커밋: `b55ceb1fdd84a813d77de09a06af83d323a81f85`
- 커밋 제목: `Merge pull request #191 from Autumn-27/feat/judge-token-usage`
- 한국어 해설판 작성일: 2026-10-05 (한국 시간)
- 복사 범위: 위 커밋에서 추적하는 전체 파일. GitHub의 Issues, Pull Requests,
  계정 설정, Secrets, 릴리스 첨부 파일은 Git 소스 파일이 아니므로 포함하지 않습니다.
- 독립 저장소는 자체 커밋 이력으로 관리합니다. 원본의 전체 과거 커밋 이력을
  가져왔다고 주장하지 않습니다. 기준 소스는 위 커밋으로 확인할 수 있습니다.

## 변경 내용

README와 학습 문서를 한국어로 작성하고 Go, TypeScript/TSX, 스크립트에
한국어 해설 주석을 추가합니다. 기능 구현, 모델 프롬프트, API 식별자와 기존 UI 문자열은
기준 소스의 동작을 유지합니다. 원문 README와 변경 기록은 `docs/upstream/`에 보존합니다.

주석은 파일의 역할, 주요 함수, 입력과 출력, 상태 변화, 동시성, 오류 처리,
테스트 의도를 설명합니다. 수정 제안과 현재 구현의 한계는 문서에서 구분하며,
해설 과정에서 제안을 이미 구현된 기능처럼 표현하지 않습니다.

## 유지한 출처·배포 식별자

`go.mod`의 모듈 경로, 내부 import, 원본 릴리스 조회 경로, Docker 이미지 이름은
동작 호환성을 위해 원본 그대로입니다. 따라서 기본 Docker Compose는
`autumn27/artex` 이미지를 가져오며, 자체 빌드가 아닙니다. 자동 업데이트도
원본 릴리스를 조회합니다. 이 저장소의 수정본을 실행하려면 README의 소스 빌드 절차를
사용하세요. 자체 릴리스·이미지 배포로 전환할 때에는 관련 경로를 함께 변경해야 합니다.

## 라이선스와 원문 고지

- 루트 [LICENSE](LICENSE): 원본 GNU Affero General Public License v3.0 전문을 보존합니다.
- [web/LICENSE](web/LICENSE): 프런트엔드에 포함된 원래의 MIT 고지를 보존합니다.
- [원문 README](docs/upstream/README.md): 원저자의 사용 제한 및 면책 고지를 포함합니다.
- [원문 변경 기록](docs/upstream/CHANGELOG.md): 원저자의 변경 설명과 기여자 표기를 보존합니다.

독립 저장소라는 사실은 원저작물의 출처·라이선스·고지 의무를 없애지 않습니다.
이 문서는 출처 및 수정 범위에 대한 기록이며 별도의 라이선스를 부여하지 않습니다.
