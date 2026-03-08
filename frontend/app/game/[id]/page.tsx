"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import ClerkMascot from "../../../components/ClerkMascot";
import Nav from "../../../components/Nav";
import { API_BASE } from "../../../components/api";

type ActionType = "buy_ingredients" | "cook" | "sell" | "rest";

export default function GamePage() {
  const params = useParams<{ id: string }>();
  const id = useMemo(() => String(params.id || ""), [params.id]);
  const [status, setStatus] = useState<any>(null);
  const [logs, setLogs] = useState<any[]>([]);
  const [msg, setMsg] = useState("");

  async function req(path: string, init?: RequestInit) {
    const res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...(init?.headers || {}) }
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error?.message || "请求失败");
    return data;
  }

  async function load() {
    const [s, h] = await Promise.all([
      req(`/api/game/${id}/status`),
      req(`/api/game/${id}/history?limit=30`)
    ]);
    setStatus(s);
    setLogs(h.items || []);
  }

  useEffect(() => {
    if (!id) return;
    load().catch((e) => setMsg(e instanceof Error ? e.message : "加载失败"));
  }, [id]);

  async function action(type: ActionType, quantity?: number) {
    if (!status) return;
    setMsg("");
    try {
      await req(`/api/game/${id}/action`, {
        method: "POST",
        body: JSON.stringify({ agent_id: "agent-demo", action_type: type, params: quantity ? { quantity } : {}, idempotency_key: `${Date.now()}-${type}` })
      });
      await load();
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "操作失败");
    }
  }

  async function nextTurn() {
    try {
      await req(`/api/game/${id}/next-turn`, { method: "POST" });
      await load();
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "失败");
    }
  }

  if (!status) return <main className="container"><p>加载中...</p><p className="small">{msg}</p></main>;

  const mood =
    status.resources.energy <= 20
      ? "tired"
      : status.resources.actions_used >= status.resources.action_quota
        ? "warn"
        : status.resources.meals > 0
          ? "good"
          : "normal";
  const line =
    status.resources.energy <= 20
      ? "体力太低了，先 rest 或 next-turn。"
      : status.resources.actions_used >= status.resources.action_quota
        ? "今天动作次数用完啦，进入下一天。"
        : status.resources.ingredients === 0
          ? "原料不够，先 buy_ingredients。"
          : status.resources.meals === 0
            ? "有原料了，快 cook。"
            : "有餐品库存，优先 sell 变现。";

  return (
    <main className="container">
      <Nav />
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>{status.game.shop_name} · Day {status.game.turn}</h1>
        <Link href="/">返回首页</Link>
      </div>

      <section className="card" style={{ marginBottom: 12 }}>
        <div className="grid grid-3">
          <div><strong>金币</strong><div>{status.resources.gold}</div></div>
          <div><strong>体力</strong><div>{status.resources.energy}</div></div>
          <div><strong>原料</strong><div>{status.resources.ingredients}</div></div>
          <div><strong>餐品</strong><div>{status.resources.meals}</div></div>
          <div><strong>经验</strong><div>{status.resources.xp}</div></div>
          <div><strong>动作</strong><div>{status.resources.actions_used}/{status.resources.action_quota}</div></div>
        </div>
        <p className="small">建议动作：{(status.suggested_actions || []).join(" / ")}</p>
      </section>

      <ClerkMascot mood={mood} line={line} />

      <section className="card" style={{ marginBottom: 12 }}>
        <h3>操作</h3>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <button onClick={() => action("buy_ingredients", 5)}>buy_ingredients x5</button>
          <button onClick={() => action("cook", 3)}>cook x3</button>
          <button onClick={() => action("sell", 3)}>sell x3</button>
          <button onClick={() => action("rest")}>rest</button>
          <button className="secondary" onClick={nextTurn}>next-turn</button>
        </div>
        <p className="small">单回合最多 2 步，动作间冷却 1 秒。</p>
        {msg ? <p style={{ color: "#b94a48" }}>{msg}</p> : null}
      </section>

      <section className="card">
        <h3>日志</h3>
        {logs.map((it, idx) => (
          <div key={`${it.ts}-${idx}`} className="log-item">
            <strong>{it.title}</strong>
            <div className="small">{new Date(it.ts).toLocaleString()}</div>
            <div>{it.detail}</div>
          </div>
        ))}
      </section>
    </main>
  );
}
