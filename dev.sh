#!/usr/bin/env bash
# [한국어 해설] 개발 실행
# 백엔드와 Next 개발 서버를 같은 프로세스 그룹에서 시작하고 wait로 유지합니다. /api 요청은 프런트에서 백엔드로 전달됩니다.
# cleanup은 종료 신호에서 그룹의 자식 프로세스를 정리합니다. 이 파일은 프로덕션 바이너리 생성 또는 개별 에이전트 격리 도구가 아닙니다.
# 开发模式：后端(:8787) + 流量代理(:8788) 与 前端 next dev(:5173) 一起跑。
# 前端 /api 反代到后端；Ctrl-C 一并退出。
#
# 单二进制（前端内嵌）方式见 README「单二进制」一节，不走这个脚本。
set -euo pipefail
cd "$(dirname "$0")"

# 退出时结束本进程组内的所有子进程（后端 + 前端）。
# [한국어 흐름] 현재 프로세스 그룹의 백엔드/프런트 자식을 종료합니다. EXIT·INT·TERM에서 호출됩니다.
cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

# 后端（普通 go run，不内嵌前端）；并发 work agent 数在「系统设置」里配置。
go run ./cmd/artex -addr :8787 -proxy 127.0.0.1:8788 &

# 前端热更新（Vite/Next dev server，/api 反代到 :8787）。
( cd web && npm run dev ) &

echo "[dev] 后端 :8787 / 代理 :8788 / 前端 http://localhost:5173  (Ctrl-C 退出)"
wait
