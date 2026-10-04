"use client";

/* 한국어 해설: 알림 설정 입력과 필터 요약
 * 필드 정의를 받아 텍스트·비밀번호·숫자·여러 줄 입력 중 알맞은 위젯을 만든다.
 * 서버가 반환한 마스킹 표식을 실제 비밀번호 입력값으로 넣지 않고 저장된 값이 있다는 안내로 표시한다.
 * asText는 표시용 변환이고 최종 설정 JSON의 구성은 호출하는 페이지의 저장 로직이 맡는다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { CheckIcon } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { NotificationFilter } from "@/lib/types";

// asText / inputType 是本文件内的取值辅助（与控件渲染强相关），不放在 channel-fields。
import { type FieldDef, type FieldKind, SEVERITY_OPTIONS } from "./channel-fields";

// asText 把任意配置值渲染成输入框可用的字符串。
// config 来自 JSON，值可能是 string / number / boolean / array / null，
// 这里只关心「能不能塞进文本框」，具体序列化由 buildConfig 负责。
function asText(v: unknown): string {
  if (typeof v === "string") return v;
  if (v === null || v === undefined) return "";
  return String(v);
}

// inputType 把字段类型映射到 input 的 type 属性。
function inputType(kind: FieldKind): "text" | "password" | "number" {
  if (kind === "password") return "password";
  if (kind === "number") return "number";
  return "text";
}

// ConfigField 按字段定义渲染对应的控件。
//
// 掩码字段的处理是这里唯一的讲究：输入框**不显示**掩码值本身，只显示一行
// 「已保存」提示。这样界面上就只有一个规则——框里有字就是用户填的，
// 空框就是空值。若把 "__masked__:…abc123" 塞进输入框，用户会以为那是要自己
// 删掉的占位文本，反而更容易误清凭据。
/* 한국어 흐름: ConfigField
 * 서버의 마스킹 값을 비밀번호 input의 실제 값으로 넣지 않는다. 기존 값 유지 여부와 새 입력을 분리해 사용자가 비밀값 표식을 지우다가 저장된 인증 정보를 의도치
 * 않게 바꾸지 않도록 한다.
 */
export function ConfigField({
  def,
  value,
  isSecret,
  onChange,
}: {
  def: FieldDef;
  value: unknown;
  isSecret: boolean;
  onChange: (v: unknown) => void;
}) {
  const id = `n-cfg-${def.key}`;
  const raw = asText(value);
  // 后端回显的掩码值：形如 "__masked__:…abc123"，尾部是原值的可辨识片段。
  const masked = isSecret && raw.startsWith("__masked__");
  const maskedTail = masked ? (raw.split("…")[1] ?? "") : "";

  if (def.kind === "switch") {
    return (
      <div className="flex items-center gap-2 text-sm">
        <Switch checked={value === true} onCheckedChange={onChange} aria-label={def.label} />
        {def.label}
        {def.help && <span className="text-muted-foreground">（{def.help}）</span>}
      </div>
    );
  }

  if (def.kind === "select") {
    return (
      <div className="grid gap-2">
        <Label>{def.label}</Label>
        <Select value={raw || def.options?.[0]?.value} onValueChange={onChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(def.options ?? []).map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    );
  }

  // 控件按字段类型分派。用 if 链而不是嵌套三元，是因为这里要区分四种控件，
  // 三层三元读起来已经要停下来数括号了。
  function control() {
    if (def.kind === "textarea" || def.kind === "kv") {
      return (
        <Textarea
          id={id}
          className="font-mono"
          placeholder={def.placeholder}
          value={masked ? "" : raw}
          onChange={(e) => onChange(e.target.value)}
        />
      );
    }
    if (def.kind === "list") {
      return (
        <Input
          id={id}
          value={Array.isArray(value) ? (value as string[]).join(", ") : raw}
          onChange={(e) => onChange(e.target.value)}
          placeholder={def.placeholder}
        />
      );
    }
    return (
      <Input
        id={id}
        className={def.kind === "text" ? "font-mono" : ""}
        type={inputType(def.kind)}
        placeholder={def.placeholder}
        value={masked ? "" : raw}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }

  const hint = masked ? (
    <p className="text-muted-foreground flex items-center gap-1 text-xs">
      <CheckIcon className="size-3" />
      已保存{maskedTail ? `（尾号 ${maskedTail}）` : ""} · 填入新值即覆盖，清空则删除该项
    </p>
  ) : (
    def.help && <p className="text-muted-foreground text-xs">{def.help}</p>
  );

  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>{def.label}</Label>
      {control()}
      {hint}
    </div>
  );
}

// FilterSummary 把过滤条件摘要成一行，让卡片不用展开就能看出这个渠道推什么。
export function FilterSummary({ filter }: { filter: NotificationFilter }) {
  const parts: string[] = [];
  if (filter.min_severity) {
    parts.push(SEVERITY_OPTIONS.find((o) => o.value === filter.min_severity)?.label ?? filter.min_severity);
  }
  if (filter.vulnclass_include?.length) parts.push(`类型含 ${filter.vulnclass_include.length} 词`);
  if (filter.vulnclass_exclude?.length) parts.push(`排除 ${filter.vulnclass_exclude.length} 词`);
  if (filter.task_ids?.length) parts.push(`${filter.task_ids.length} 个任务`);
  if (filter.asset_ids?.length) parts.push(`${filter.asset_ids.length} 个资产`);
  if (filter.on_status_change) parts.push("含状态变更");
  if (parts.length === 0) {
    return <p className="text-muted-foreground text-sm">全部漏洞</p>;
  }
  return <p className="text-muted-foreground text-sm">{parts.join(" · ")}</p>;
}
