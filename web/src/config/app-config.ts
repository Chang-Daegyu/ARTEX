/**
 * 한국어 해설 — src/config/app-config.ts
 * 사이트 이름·메타 태그·저작권 표시의 공통 설정.
 * 웹 패키지 버전은 package.json에서 읽으며, Go 바이너리의 실제 배포 버전은 api.health가 별도로 조회한다.
 * 현재 연도는 모듈 초기화 시 계산된다. 문자열은 원본 UI 및 표시 계약을 유지한다.
 */

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

export const APP_CONFIG = {
  name: "ARTEX",
  version: packageJson.version,
  copyright: `© ${currentYear}, ARTEX.`,
  meta: {
    title: "ARTEX — 自主渗透测试控制台",
    description: "LLM 驱动的自主渗透测试系统控制台",
  },
};
