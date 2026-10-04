/**
 * 한국어 해설 — postcss.config.mjs
 * CSS 빌드 시 Tailwind CSS의 PostCSS 플러그인을 등록한다.
 * UI의 유틸리티 클래스와 테마 변수를 최종 스타일시트로 변환하는 진입점이다.
 * 실제 색상·레이아웃 값은 src/app/globals.css와 src/styles/presets를 함께 읽는다.
 */

/** @type {import('postcss-load-config').Config} */
const config = {
  plugins: {
    "@tailwindcss/postcss": {},
  },
};

export default config;
