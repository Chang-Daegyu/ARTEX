/**
 * 한국어 해설 — next.config.mjs
 * Next.js 빌드·개발 서버의 실행 방식 선택.
 * NEXT_EXPORT=1이면 web/out에 정적 HTML·JS를 생성하고, NEXT_PUBLIC_MOCK=1이면 데모용 데이터 경로를 사용한다.
 * 그 외 실행에서는 일반 API를 Go 서버로 프록시한다. SSE 접속 주소 선택은 src/lib/api.ts에 별도로 있다.
 * 정적 배포의 백엔드·인증·LLM 실행 주체는 Go 서버이며, 이 설정은 해당 기능을 브라우저로 옮기지 않는다.
 */

import { fileURLToPath } from "node:url";

// 静态导出：`NEXT_EXPORT=1 next build` 产出纯静态目录到 web/out，可直接丢进
// nginx web 根目录运行。开发(next dev)不设该变量，保留 /api 反代与热更新。
const isExport = process.env.NEXT_EXPORT === "1";
// Vercel demo：整站走 mock，无后端，无需 /api 反代。
const isMock = process.env.NEXT_PUBLIC_MOCK === "1";

/** @type {import('next').NextConfig} */
const nextConfig = {
  // 避免父目录的 lockfile 影响根目录推断及资源路径生成。
  turbopack: { root: fileURLToPath(new URL(".", import.meta.url)) },
  reactCompiler: true,
  // 允许从局域网 IP 访问 dev 资源（HMR），按需增删。
  // dev 阶段放开任意 IPv4 来源访问 /_next/* 与 HMR（局域网 IP 变动也不受影响）。
  // 注意：Next 出于安全禁止裸 "*"，需用分段通配；"*.*.*.*" 匹配任意 IPv4。
  allowedDevOrigins: ["*.*.*.*"],
  compiler: {
    removeConsole: process.env.NODE_ENV === "production",
  },
  ...(isExport
    ? {
        // 纯静态导出：无 Node 运行时；图片不经优化；每个路由产出 <route>/index.html。
        output: "export",
        images: { unoptimized: true },
        trailingSlash: true,
      }
    : isMock
      ? {
          // Vercel mock demo：无后端，不需要 /api 反代。
          images: { unoptimized: true },
        }
      : {
          // 开发：把 /api/* 反代到 Go 后端（默认 :8787，可用 AUTOPENTEST_API 覆盖）。
          async rewrites() {
            const backend = process.env.AUTOPENTEST_API ?? "http://localhost:8787";
            return [{ source: "/api/:path*", destination: `${backend}/api/:path*` }];
          },
        }),
};

export default nextConfig;
