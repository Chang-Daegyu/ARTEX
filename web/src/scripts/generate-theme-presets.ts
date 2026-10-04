/**
 * 한국어 해설 — src/scripts/generate-theme-presets.ts
 * CSS 테마 메타데이터에서 TypeScript 프리셋 목록을 생성하는 개발 스크립트.
 * 각 preset의 이름/키와 밝은·어두운 primary 색을 읽고 globals.css의 기본 테마를 앞에 추가한다.
 * 정해진 생성 구역만 치환하고 Biome으로 포맷한 결과가 달라질 때 theme.ts에 저장한다.
 * 현재 저장소의 자동 실행 연결은 web/.husky/pre-commit이다. 아래 원문 주석의 pre-push 표기는 현재 훅 파일과 다르다.
 * CSS 메타데이터와 생성 시작/끝 표시는 파서 입력이므로 설명 주석을 추가할 때도 그대로 유지한다.
 */

/**
 * Script: generate-theme-presets.ts
 *
 * This script scans the /styles/presets directory for CSS files containing theme definitions.
 * It extracts `label:`, `value:`, and primary color definitions (`--primary`) for both light and dark modes.
 * These primary colors are used to visually represent each theme in the UI (e.g., colored dots or theme previews).
 * Default theme colors are fetched from /app/globals.css.
 * All extracted metadata is injected into a marked section of the /lib/preferences/theme.ts file.
 *
 * Usage:
 * - During local development, run manually after adding any new theme preset:
 *     npm run generate:presets
 * - Ensure that each new CSS preset includes `label:` and `value:` comments.
 * - This generation step is currently automated using a Husky pre-push hook.
 * - You may optionally integrate it directly into a build step if preferred.
 */

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const presetDir = path.resolve(__dirname, "../styles/presets");

if (!fs.existsSync(presetDir)) {
  console.error(`❌ Preset directory not found at: ${presetDir}`);
  process.exit(1);
}

const outputPath = path.resolve(__dirname, "../lib/preferences/theme.ts");

const files = fs.readdirSync(presetDir).filter((file) => file.endsWith(".css"));

if (files.length === 0) {
  console.warn("⚠️ No preset CSS files found. Only default preset will be included.");
}

// CSS 파일의 메타데이터와 두 모드의 대표색을 읽는다. 파일 내용 중 첫 정규식 일치를 사용한다.
const presets = files.map((file) => {
  const filePath = path.join(presetDir, file);
  const content = fs.readFileSync(filePath, "utf8");

  const labelMatch = content.match(/label:\s*(.+)/);
  const valueMatch = content.match(/value:\s*(.+)/);

  if (!labelMatch) {
    console.warn(`⚠️ No 'label:' found in ${file}, using filename as fallback.`);
  }
  if (!valueMatch) {
    console.warn(`⚠️ No 'value:' found in ${file}, using filename as fallback.`);
  }

  const label = labelMatch?.[1]?.trim() ?? file.replace(".css", "");
  const value = valueMatch?.[1]?.trim() ?? file.replace(".css", "");

  const lightPrimaryMatch = content.match(/:root\[data-theme-preset="[^"]*"\][\s\S]*?--primary:\s*([^;]+);/);
  const darkPrimaryMatch = content.match(/\.dark:root\[data-theme-preset="[^"]*"\][\s\S]*?--primary:\s*([^;]+);/);

  const primary = {
    light: lightPrimaryMatch?.[1]?.trim() ?? "",
    dark: darkPrimaryMatch?.[1]?.trim() ?? "",
  };

  if (!lightPrimaryMatch || !darkPrimaryMatch) {
    console.warn(`⚠️ Missing --primary for ${file} (light or dark). Check CSS syntax.`);
  }

  return { label, value, primary };
});

const globalStylesPath = path.resolve(__dirname, "../app/globals.css");

let globalContent = "";
try {
  globalContent = fs.readFileSync(globalStylesPath, "utf8");
} catch (err) {
  console.error(`❌ Could not read globals.css at ${globalStylesPath}`);
  console.error(err);
  process.exit(1);
}

const defaultLightPrimaryRegex = /:root\s*{[^}]*--primary:\s*([^;]+);/;
const defaultDarkPrimaryRegex = /\.dark\s*{[^}]*--primary:\s*([^;]+);/;

const defaultLightPrimaryMatch = defaultLightPrimaryRegex.exec(globalContent);
const defaultDarkPrimaryMatch = defaultDarkPrimaryRegex.exec(globalContent);

const defaultPrimary = {
  light: defaultLightPrimaryMatch?.[1]?.trim() ?? "",
  dark: defaultDarkPrimaryMatch?.[1]?.trim() ?? "",
};

presets.unshift({ label: "Default", value: "default", primary: defaultPrimary });

const generatedBlock = `// --- generated:themePresets:start ---

export const THEME_PRESET_OPTIONS = ${JSON.stringify(presets, null, 2)} as const;

export const THEME_PRESET_VALUES = THEME_PRESET_OPTIONS.map((p) => p.value);

export type ThemePreset = (typeof THEME_PRESET_OPTIONS)[number]["value"];

// --- generated:themePresets:end ---`;

const fileContent = fs.readFileSync(outputPath, "utf8");

// 표시된 생성 구역만 바꾸어 이 파일 바깥의 수동 설명·모드 정의를 보존한다.
const updated = fileContent.replace(
  /\/\/ --- generated:themePresets:start ---[\s\S]*?\/\/ --- generated:themePresets:end ---/,
  generatedBlock,
);

// Biome 실행 파일을 명시적으로 찾아 stdin으로 생성 결과를 포맷한다.
// 기존 내용과 같으면 쓰지 않고 바뀐 경우에만 출력 파일을 덮어써 불필요한 변경을 줄인다.
function main() {
  const biomeBin = require.resolve("@biomejs/biome/bin/biome");
  const formatted = execFileSync(process.execPath, [biomeBin, "format", "--stdin-file-path", outputPath], {
    input: updated,
    encoding: "utf8",
  });

  if (formatted === fileContent) {
    console.log("ℹ️  No changes in theme.ts");
    return;
  }

  fs.writeFileSync(outputPath, formatted);
  console.log("✅ theme.ts updated with new theme presets");
}

try {
  main();
} catch (err) {
  console.error("❌ Unexpected error while generating theme presets:", err);
  process.exit(1);
}
