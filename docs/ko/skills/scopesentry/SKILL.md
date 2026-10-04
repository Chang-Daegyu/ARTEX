> **사람이 읽기 위한 한국어 번역·해설입니다. ARTEX가 이 파일을 Skill로 자동 로드하지 않습니다.** 실행용 원문은 [skills/scopesentry/SKILL.md](../../../../skills/scopesentry/SKILL.md)에 그대로 보존되어 있습니다. 아래 내용은 저장소에 포함된 원문의 사용 안내이며, 이 번역 작업에서 실제 ScopeSentry 인스턴스에 접속하거나 스캔·키 생성·설정 변경을 수행하지 않았습니다.

# ScopeSentry MCP 사용 안내

원문이 붙인 이름은 `scopesentry-mcp`다. 설명은 “ScopeSentry MCP를 통해 프로젝트·작업·템플릿·자산·노드를 관리한다. ScopeSentry, MCP, API Key, 스캔 작업, 자산 조회를 다룰 때 사용한다”는 의미다. 저장소의 디렉터리 이름은 `scopesentry`이므로 문서 안의 이름과 구분한다.

이 안내는 **이미 배포된 ScopeSentry 인스턴스**를 가진 사용자를 대상으로 한다. Cursor 또는 다른 MCP 클라이언트로 플랫폼에 연결하며, 사용자의 컴퓨터에 ScopeSentry 소스가 있어야 하는 방식은 아니다. 도구의 실제 매개변수는 연결된 MCP 서버가 제공하는 schema를 기준으로 확인한다.

## 1. 준비

### 1.1 서비스 주소 확인

- 원문에서 제시하는 기본 웹 화면은 `http://<호스트>`다.
- MCP 엔드포인트는 `http://<호스트>/mcp`다. 앞에 역방향 프록시나 프런트 프록시가 있으면 외부에서 접근 가능한 실제 `/mcp` 주소를 사용한다.
- 아래 예시의 `8082`는 예시 포트다. 실제 배포 포트와 프록시 구성을 먼저 확인한다.

### 1.2 API Key 만들기

1. 브라우저에서 ScopeSentry 웹 화면에 로그인한다.
2. **API Key** 관리 화면 또는 관리자가 제공한 인터페이스에서 키를 만든다.
3. 반환된 `ssk_...` 문자열을 저장한다. 원문은 이 문자열이 **한 번만 표시된다**고 설명한다.

### 1.3 Cursor MCP 구성

원문의 경로는 Cursor → Settings → MCP → 서버 추가다. 실제 클라이언트 UI 위치는 버전에 따라 확인한다. `url`에는 해당 인스턴스의 MCP 주소, `X-API-Key`에는 발급받은 키를 넣는다. 예시 안의 중국어 값은 모두 호스트와 키를 나타내는 자리표시자다.

```json
{
  "mcpServers": {
    "scopesentry": {
      "url": "http://<你的主机>:8082/mcp",
      "headers": {
        "X-API-Key": "ssk_你的密钥"
      }
    }
  }
}
```

대체 헤더 형식은 `Authorization: Bearer ssk_<발급받은_키>`다. 구성 후 MCP 연결을 다시 시작하거나 클라이언트를 다시 불러와 `list_projects`, `list_assets` 등이 나타나는지 확인한다.

## 2. 도구 전체 목록

| 도구 | 한국어 의미와 역할 |
| --- | --- |
| `list_projects` | 태그로 묶은 프로젝트 트리와 프로젝트 ID 조회 |
| `list_projects_data` | 이름 검색을 지원하는 프로젝트 페이지 목록 |
| `get_project` | 프로젝트 상세 |
| `create_project` | 프로젝트 생성 |
| `list_tasks` | 스캔 작업 목록 |
| `get_task` | 작업 상세 |
| `list_scan_templates` | 스캔 템플릿 목록 |
| `get_scan_template` | 템플릿 상세 |
| `list_plugin_modules` | 스캔 단계의 모듈 이름 목록 |
| `list_plugins` | 플러그인의 hash와 기본 parameter를 포함한 목록 |
| `create_scan_template` | 스캔 템플릿 생성 |
| `create_scan_task` | 스캔 작업 생성 |
| `list_assets` | 자산 종류별 페이지 조회 |
| `count_assets` | 자산 개수 집계; `/api/assets/common/total`에 대응 |
| `get_asset_detail` | 자산 또는 취약점 상세 |
| `add_asset_tag` | 자산에 태그 추가 |
| `list_nodes` | 스캔 노드 목록 |

각 도구의 매개변수는 MCP 도구 설명과 schema를 따른다. 특히 `list_assets`와 `count_assets`는 같은 `search`·`filter` 문법을 사용하므로 자산 조회 전에 `list_assets`의 description을 읽으면 된다.

총개수만 필요하면 `count_assets`를 사용한다. 전체 페이지를 반복해서 가져온 뒤 행을 세는 대신 웹 화면의 총개수 조회와 같은 집계 경로를 사용하는 것이다.

## 3. 자주 사용하는 흐름

### 3.1 프로젝트를 정해 자산 조회

사용자 요청이나 현재 맥락에 프로젝트가 있으면 `filter.project`로 먼저 범위를 줄인다. 여러 프로젝트의 결과가 섞이는 일과 불필요하게 큰 응답을 줄이기 위한 지침이다. 프로젝트 조건이 주어지지 않았다면 임의로 프로젝트 하나를 강제할 필요는 없다.

1. `list_projects` 또는 `list_projects_data`로 대상 프로젝트의 **ObjectID**를 찾는다. 결과의 `id` 또는 `children[].value`를 확인한다.
2. `list_assets`의 `filter.project`에 그 ID를 넣는다. **프로젝트 표시 이름을 넣는 필드가 아니다.**

다음 예시는 `asset` 종류에서 도메인 접두 조건과 프로젝트 ID를 함께 적용한다. `<项目ObjectID>`는 `<프로젝트ObjectID>`라는 뜻이다.

```json
{
  "asset_type": "asset",
  "pageIndex": 1,
  "pageSize": 20,
  "search": "domain=^example.com",
  "filter": {
    "project": ["<项目ObjectID>"]
  }
}
```

### 3.2 스캔 작업 생성

1. `list_nodes`로 현재 온라인인 노드 이름을 확인한다.
2. `list_scan_templates`에서 사용할 템플릿의 ObjectID를 찾거나 `create_scan_template`로 만든다.
3. `create_scan_task`에 필수값 `name`, `node`를 넣는다. `template`은 템플릿 이름이 아니라 **템플릿 ObjectID**다.

`targetSource`는 대상을 어디서 가져오는지 나타내며 원문은 웹 화면과 같은 의미라고 설명한다.

| `targetSource` | 대상 출처 | 필요한 매개변수 |
| --- | --- | --- |
| `general` | 대상을 직접 입력 | `target` |
| `project` | 프로젝트에서 읽기 | 프로젝트 ObjectID 배열 `project` |
| `asset` | 웹 자산 목록 검색 | `search`; 선택적으로 `project`, `filter`, `targetNumber` |
| `RootDomain` | 루트 도메인 목록 검색 | `search`; 선택적으로 `project`, `filter`, `targetNumber` |
| `subdomain` | 하위 도메인 목록 검색 | `search`; 선택적으로 `project`, `filter`, `targetNumber` |
| `UrlScan` | URL 스캔 결과 검색 | `search`; 선택적으로 `project`, `filter`, `targetNumber` |
| `*Source`, 예: `subdomainSource` | 자산 화면에서 선택/검색한 대상으로 생성 | `targetTp=search`면 `search`, `targetTp=select`면 `targetIds` |

직접 루트 도메인을 지정하는 원문 예시는 다음과 같다. `target`의 줄바꿈으로 여러 도메인을 구분한다. `example-子域名收集`는 “example 하위 도메인 수집”이라는 작업 이름이다.

```json
{
  "name": "example-子域名收集",
  "node": ["node-1"],
  "template": "<模板ObjectID>",
  "targetSource": "general",
  "target": "example.com\nfoo.com",
  "project": ["<项目ObjectID>"]
}
```

다음 단계는 이전 작업 이름으로 하위 도메인 목록을 검색해 후속 모듈을 실행하는 예시다. 검색 필드 `task`는 ID가 아니라 **작업 이름**이다. `example-端口与漏洞`는 “example 포트와 취약점”, `<后续模块模板ObjectID>`는 후속 모듈 템플릿의 ObjectID를 뜻한다.

```json
{
  "name": "example-端口与漏洞",
  "node": ["node-1"],
  "template": "<后续模块模板ObjectID>",
  "targetSource": "subdomain",
  "search": "task==\"example-子域名收集\"",
  "project": ["<项目ObjectID>"]
}
```

### 3.3 루트 도메인의 전체 정보 수집: 두 단계로 나누기

원문은 루트 도메인에서 폭넓은 정보 수집을 할 때 한 번에 모든 단계를 실행하기보다 두 작업으로 나누는 방식을 권한다. 이유는 분산 작업이 **개별 목표 하나**를 단위로 배분되기 때문이다. 한 노드가 루트 도메인 하나를 맡으면 그 노드에서 찾은 하위 도메인의 후속 모듈도 같은 노드에 몰릴 수 있다. 따라서 하위 도메인을 먼저 저장한 뒤, 저장된 하위 도메인 각각을 새 목표로 나누면 배분 단위가 더 잘게 나뉜다.

| 단계 | 대상 출처 | 템플릿과 조건 | 결과 |
| --- | --- | --- | --- |
| 1: 하위 도메인 수집 | `targetSource=general`, 여러 줄의 루트 도메인 `target` | `SubdomainScan`, `SubdomainSecurity`만 활성화 | `get_task`로 완료를 확인하고 하위 도메인을 저장 |
| 2: 후속 정보 수집 | `targetSource=subdomain` | `search: task=="<1단계 작업 이름>"`; 필요 시 프로젝트 제한; 포트·자산·취약점 등 후속 모듈 | 각 하위 도메인을 독립 목표로 여러 노드에 분배 |

웹의 하위 도메인 자산 화면에서 작업 이름으로 검색한 뒤 “하위 도메인에서 작업 생성”을 사용해도 같은 취지다. 2단계 템플릿에는 이미 완료한 `SubdomainScan`을 다시 넣지 않을 수 있다. 이 설계의 학습 포인트는 **발견 단계의 결과를 저장하고 다음 단계의 분산 처리 단위로 다시 사용한다**는 점이다.

### 3.4 스캔 템플릿 생성

1. `list_plugin_modules`로 모듈 이름을 확인한다.
2. `list_plugins`를 필요 시 `module`로 필터해 플러그인의 `hash`와 기본 `parameter`를 확인한다.
3. `create_scan_template`의 `modules`에 “모듈 이름 → 플러그인 hash 배열”을 넣는다.

표시 이름과 실행 식별자를 혼동하지 않는 것이 핵심이다. 프로젝트/템플릿은 ObjectID, 실행 노드는 이름, 플러그인은 hash를 사용하는 연결이다.

## 4. 자산 조회: `list_assets`와 `count_assets`

두 도구는 같은 `asset_type`, `search`, `filter`를 받는다. `count_assets`는 `{ "total": N }`을 반환하며 원문은 이를 웹의 `/api/assets/common/total` 경로와 연결한다.

다음은 특정 작업 이름과 프로젝트에 속한 하위 도메인 수를 조회할 때 사용할 조건 예시다. `某任务名`은 “특정 작업 이름”이라는 자리표시자다.

```json
{
  "asset_type": "subdomain",
  "search": "task==\"某任务名\"",
  "filter": {"project": ["<项目ObjectID>"]}
}
```

원문의 성능 지침은 프로젝트 조건이 있으면 먼저 범위를 제한하고, 알고 있는 값에는 `==` 또는 `^`로 시작하는 접두 일치를 쓰라는 것이다. 정규식 포함 검색은 넓은 범위에서 비용이 커질 수 있다는 설명이다. 실제 인덱스 사용 여부와 성능은 배포된 서버의 인덱스·데이터 분포·정규식 형태로 확인해야 하며, 이 번역에서 쿼리 실행 계획을 측정한 것은 아니다.

### 4.1 `asset_type` 전체 값

`asset`, `RootDomain`, `subdomain`, `app`, `mp`, `UrlScan`, `SensitiveResult`, `DirScanResult`, `crawler`, `vulnerability`, `PageMonitoring`, `IPAsset`, `SubdomainTakerResult`를 제시한다. 대소문자도 식별자의 일부로 보존한다.

원문의 별칭 예시는 `web` → `asset`, `vuln` → `vulnerability`, `ip` → `IPAsset`, `url` → `UrlScan`이다.

### 4.2 공통 매개변수

| 매개변수 | 의미 |
| --- | --- |
| `pageIndex`, `pageSize` | 페이지 번호와 크기; 원문 기본값은 1, 20 |
| `search` | 다음 절의 검색 식 |
| `filter` | 다음 절의 정확한 값 필터 JSON |
| `sort` | `UrlScan`, `DirScanResult`에서만 `length` 정렬 지원 |
| `sid` | `SensitiveResult`에서만 사용하는 민감정보 규칙 이름 |

`search`와 `filter`를 동시에 사용할 수 있다.

### 4.3 `search` 검색 식

자체 DSL이며 SQL 문법이 아니다.

| 연산자 | 의미 | 원문 인덱스 설명 | 예시 |
| --- | --- | --- | --- |
| `=` | 정규식 기반 부분 일치 | 일반 포함 검색은 인덱스 활용이 어렵다고 설명 | `domain=example` |
| `==` | 완전 일치 | 인덱스를 사용하는 방식으로 제시 | `port==443` |
| `!=` | 제외 | 별도 보장 없음 | `port!="80"` |
| `&&` | AND | 조건 결합 | `domain==example.com && port==443` |
| `\|\|` | OR | 조건 결합 | `title=admin \|\| body=login` |

원문은 `domain`, `ip`, `port`, `title` 등에 인덱스가 있으며 완전 일치 또는 `domain=^example.com` 같은 접두 정규식이 성능에 유리하다고 설명한다. `=`를 무조건 같은 성능의 문자열 비교로 이해하면 안 된다.

모든 자산 종류에 공통으로 제시하는 검색 필드는 `tag`, `task`, `rootDomain`이다. `task`는 작업 이름이다. **프로젝트는 `search` 안에 쓰지 않고 `filter.project`에 넣는다.** 원문은 `search`의 project 조건이 무효이거나 `&&`와 결합하면 오류가 될 수 있다고 명시한다.

| `asset_type` | 자주 사용하는 `search` 필드 |
| --- | --- |
| `asset` | domain, ip, port, service, app, title, statuscode, icon, banner, type, body, header |
| `RootDomain` | domain, icp, company |
| `subdomain` | domain, ip, type, value |
| `app` | name, icp, company, category, description, url, apk |
| `mp` | name, icp, company, category, description, url |
| `UrlScan` | url, input, source, resultId, type |
| `SensitiveResult` | url, sname, body, info, md5 |
| `DirScanResult` | url, statuscode, redirect, length |
| `vulnerability` | url, vulname, matched, request, response, level |
| `crawler` | url, method, body, resultId |
| `PageMonitoring` | url, hash, diff, response |
| `IPAsset` | ip, domain, port, service, webServer, app |
| `SubdomainTakerResult` | domain, value, type, response |

예시는 다음과 같이 해석한다.

- `domain==www.example.com && port==443`: 해당 도메인과 포트의 완전 일치.
- `domain=^example.com`: 해당 문자열로 시작하는 도메인.
- `ip==192.168.1.1`: IP 완전 일치.
- `task=="어떤 작업 이름"`: 작업 이름 완전 일치.
- `level==high`: `vulnerability`의 높은 등급.
- `statuscode==200`: `DirScanResult`의 HTTP 200 결과.
- `title=admin`: 제목에 대한 부분 검색. 원문은 프로젝트 등으로 범위를 줄여 사용하라고 안내한다.

### 4.4 `filter` 정확한 값 필터

JSON의 같은 키에 여러 값이 있으면 **OR**, 서로 다른 키는 **AND**로 연결한다. 프로젝트 조건이 이미 있고 해당 종류가 이를 지원하면 `project`를 넣어 범위를 줄인다. 프로젝트 정보가 없으면 강제로 만들 필요는 없다.

| 필터 키 | 한국어 의미 | 값의 기준 |
| --- | --- | --- |
| `project` | 소속 프로젝트 | `list_projects` / `list_projects_data`의 **ObjectID** |
| `task` | 출처 작업 | `list_tasks`의 **name** |
| `port` | 포트 | 예: `"443"` |
| `service` | 서비스/프로토콜 | 예: `"https"` |
| `app` | 앱 지문 | 예: `"Nginx"` |
| `icon` | 아이콘 hash | hash 값 |
| `statuscode` | HTTP 상태 코드 | 주로 `asset` |
| `status` | 상태 | URL/디렉터리 스캔의 HTTP 코드 또는 취약점·민감정보 처리 상태 |
| `level` | 취약점 등급 | critical, high, medium, low, info |
| `type` | 자산 종류별 타입 | 예: 하위 도메인의 A, CNAME |
| `color` | 민감정보 규칙 색 | `SensitiveResult` |
| `sname` | 민감정보 규칙 이름 | `SensitiveResult` |
| `tags` | 태그 | 해당 태그 값 |

지원하는 필터 키가 종류마다 다르므로 다음 표를 기준으로 구분한다.

| `asset_type` | 사용 가능한 필터 키 |
| --- | --- |
| `asset` | project, port, service, app, icon, statuscode, type, task, tags |
| `RootDomain` | project, tags |
| `subdomain` | project, type, task, tags |
| `app` / `mp` | project, tags |
| `UrlScan` | status, tags |
| `DirScanResult` | status, tags |
| `SensitiveResult` | status, color, sname, tags |
| `crawler` | project, task, tags |
| `vulnerability` | project, level, status, task, tags |
| `PageMonitoring` / `SubdomainTakerResult` | tags |
| `IPAsset` | project, port, service, app |

프로젝트와 포트를 조합하는 원문 예시다.

```json
{"project": ["<项目ObjectID>"], "port": ["443"]}
```

검색 식과 필터, 페이지 크기를 함께 사용하는 예시는 다음과 같다. 이 값들은 원문 예시를 보존한 것이며 실제 대상에 요청하지 않았다.

```json
{
  "asset_type": "asset",
  "search": "domain=^baidu && port==443",
  "filter": {"project": ["<项目ObjectID>"]},
  "pageIndex": 1,
  "pageSize": 10
}
```

기억할 구분은 다음과 같다.

- `filter.project`에는 표시 이름 대신 ObjectID를 넣는다.
- 값 전체를 알면 `==`, 접두 조건은 `^`, 포함 검색이 필요할 때만 `=`를 사용한다.
- `UrlScan`의 HTTP 상태는 `filter.status`다.
- `DirScanResult`는 `search`의 `statuscode==200`도 사용할 수 있다.
- `SensitiveResult`의 규칙 이름은 `search`의 `sname=<규칙명>` 또는 `filter.sname`이다.

### 4.5 `sort`

원문은 `UrlScan`과 `DirScanResult`에서만 다음 `length` 정렬을 지원한다고 설명한다.

```json
{"length": "ascending"}
```

다른 종류는 `sort`를 무시하고 기본 시간순 정렬을 사용한다는 설명이다. 실제 연결 서버의 schema가 달라졌다면 그 명세를 우선 확인한다.

## 5. 스캔 템플릿의 모듈 이름

다음 식별자를 원문 그대로 사용한다.

| 모듈 | 이름으로 이해할 수 있는 단계 |
| --- | --- |
| `TargetHandler` | 입력 목표 처리 |
| `SubdomainScan` | 하위 도메인 수집 |
| `SubdomainSecurity` | 하위 도메인 보안 확인 |
| `PortScanPreparation` | 포트 검사 준비 |
| `PortScan` | 포트 검사 |
| `PortFingerprint` | 포트/서비스 지문 |
| `AssetMapping` | 자산 식별·매핑 |
| `AssetHandle` | 자산 후처리 |
| `URLScan` | URL 검사 |
| `WebCrawler` | 웹 크롤링 |
| `URLSecurity` | URL 관련 보안 검사 |
| `DirScan` | 경로 검사 |
| `VulnerabilityScan` | 취약점 검사 |
| `PassiveScan` | 수동적 관찰 기반 검사 |

오른쪽은 명칭을 이해하기 위한 한국어 풀이이며 각 모듈에서 실제 어떤 플러그인이 실행되는지는 `list_plugins`와 템플릿의 hash 배열을 확인해야 한다.

## 6. 문제 해결

| 증상 | 원문이 권하는 확인 |
| --- | --- |
| MCP에 도구가 보이지 않음 | URL, API Key, ScopeSentry 실행 여부 |
| 401 또는 403 | API Key 유효성과 교체 필요 여부 |
| 자산이 조회되지 않음 | `filter.project`가 ObjectID인지, `search`에 project를 넣지 않았는지 |
| 템플릿/작업 생성 실패 | `template`에 템플릿 ObjectID, `node`에 온라인 노드 이름을 넣었는지 |
| 조회가 느리거나 멈춘 것 같음 | 가능한 프로젝트 제한, 인덱스에 맞는 `==`/접두 검색, 과도한 부분 검색 감소, 작은 `pageSize` |

## 번역 범위와 원문 대응

원문의 준비 1.1–1.3, 도구 목록, 작업 흐름 3.1–3.4, 자산 조회 4.1–4.5, 전체 모듈 이름, 문제 해결 표를 모두 한국어로 설명했다. 명령·API 키 이름·필드·자산 타입·모듈 식별자는 보존했다. 두 단계 실행을 나타낸 원문 그림은 동일한 의존 관계를 표로 풀어 설명했다. 실행용 Skill 파일과 실제 연결 설정은 수정하지 않았다.
