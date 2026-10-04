/**
 * 한국어 해설 — src/lib/fonts/registry.ts
 * 글꼴 선택 키·표시 이름·Next.js 글꼴 변수의 연결표.
 * 현재 원본 구현은 오프라인 빌드용으로 일반 글꼴 선택을 로컬 Geist Sans, 일부 고정폭 선택을 Geist Mono에 연결한다.
 * 따라서 메뉴의 Inter/Noto 등의 이름이 각 실제 폰트 파일을 뜻하지는 않는다. Geist Pixel은 별도 모듈을 사용한다.
 * fontVars는 루트에 붙일 CSS 변수 클래스, fontOptions는 선택 UI용 배열로 같은 레지스트리에서 파생한다.
 */

// TEMPORARY: offline build fallback — uses local Geist fonts instead of Google Fonts.
// Restore registry.ts.bak after build.
import localFont from "next/font/local";
import { GeistPixelSquare } from "geist/font/pixel";

const geist = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-geist" });
const inter = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-inter" });
const notoSans = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-noto-sans" });
const roboto = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-roboto" });
const outfit = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-outfit" });
const dmSans = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-dm-sans" });
const nunitoSans = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-nunito-sans" });
const figtree = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-figtree" });
const raleway = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-raleway" });
const publicSans = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-public-sans" });
const notoSerif = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-noto-serif" });
const robotoSlab = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-roboto-slab" });
const merriweather = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-merriweather" });
const lora = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-lora" });
const playfairDisplay = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-sans/Geist-Variable.woff2", variable: "--font-playfair-display" });
const geistMono = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-mono/GeistMono-Variable.woff2", variable: "--font-geist-mono" });
const jetBrainsMono = localFont({ src: "../../../node_modules/geist/dist/fonts/geist-mono/GeistMono-Variable.woff2", variable: "--font-jetbrains-mono" });

export const fontRegistry = {
  geist: { label: "Geist", font: geist },
  inter: { label: "Inter", font: inter },
  notoSans: { label: "Noto Sans", font: notoSans },
  nunitoSans: { label: "Nunito Sans", font: nunitoSans },
  figtree: { label: "Figtree", font: figtree },
  roboto: { label: "Roboto", font: roboto },
  raleway: { label: "Raleway", font: raleway },
  dmSans: { label: "DM Sans", font: dmSans },
  publicSans: { label: "Public Sans", font: publicSans },
  outfit: { label: "Outfit", font: outfit },
  geistMono: { label: "Geist Mono", font: geistMono },
  geistPixelSquare: { label: "Geist Pixel Square", font: GeistPixelSquare },
  jetBrainsMono: { label: "JetBrains Mono", font: jetBrainsMono },
  notoSerif: { label: "Noto Serif", font: notoSerif },
  robotoSlab: { label: "Roboto Slab", font: robotoSlab },
  merriweather: { label: "Merriweather", font: merriweather },
  lora: { label: "Lora", font: lora },
  playfairDisplay: { label: "Playfair Display", font: playfairDisplay },
} as const;

export type FontKey = keyof typeof fontRegistry;

export const fontVars = (Object.values(fontRegistry) as Array<(typeof fontRegistry)[FontKey]>)
  .map((f) => f.font.variable)
  .join(" ");

export const fontOptions = (Object.entries(fontRegistry) as [FontKey, { label: string; font: { variable: string } }][]).map(
  ([key, f]) => ({
    key,
    label: f.label,
    variable: f.font.variable,
  }),
);
