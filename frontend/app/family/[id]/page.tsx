"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { API_BASE } from "../../../components/api";

type ActionType = "study" | "exercise" | "rest" | "talk" | "class" | "discipline";

interface StatusResponse {
  family: { family_id: string; name: string; generation: number; turn: number };
  resources: {
    family_money: number;
    parent_energy: number;
    child_energy: number;
    action_quota: number;
    actions_used: number;
  };
  child: {
    name: string;
    age_month: number;
    intelligence: number;
    discipline: number;
    health: number;
    stress: number;
    self_esteem: number;
    rebellion: number;
    study_score: number;
  };
  suggested_actions: ActionType[];
}

interface HistoryResponse {
  items: Array<{ ts: string; type: string; title: string; detail: string }>;
}

const actions: ActionType[] = ["study", "exercise", "rest", "talk", "class", "discipline"];

export default function FamilyPage() {
  const params = useParams<{ id: string }>();
  const id = useMemo(() => String(params.id || ""), [params.id]);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [history, setHistory] = useState<HistoryResponse["items"]>([]);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) }
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error?.message || "请求失败");
    return data as T;
  }

  async function load() {
    const [s, h] = await Promise.all([
      request<StatusResponse>(`/api/family/${id}/status`),
      request<HistoryResponse>(`/api/family/${id}/history?limit=30`)
    ]);
    setStatus(s);
    setHistory(h.items);
  }

  useEffect(() => {
    if (!id) return;
    load().catch((e) => setMessage(e instanceof Error ? e.message : "加载失败"));
    const timer = setInterval(() => {
      load().catch(() => undefined);
    }, 4000);
    return () => clearInterval(timer);
  }, [id]);

  async function doAction(action: ActionType) {
    if (!status) return;
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/family/${id}/action`, {
        method: "POST",
        body: JSON.stringify({
          agent_id: "agent-demo",
          action_type: action,
          idempotency_key: `${Date.now()}-${action}`
        })
      });
      await load();
    } catch (e) {
      setMessage(e instanceof Error ? e.message : "操作失败");
    } finally {
      setBusy(false);
    }
  }

  async function nextTurn() {
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/family/${id}/next-turn`, { method: "POST" });
      await load();
    } catch (e) {
      setMessage(e instanceof Error ? e.message : "推进失败");
    } finally {
      setBusy(false);
    }
  }

  async function nextGeneration() {
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/family/${id}/next-generation`, { method: "POST" });
      await load();
    } catch (e) {
      setMessage(e instanceof Error ? e.message : "换代失败");
    } finally {
      setBusy(false);
    }
  }

  if (!status) {
    return <main className="container"><p>加载中...</p><p className="small">{message}</p></main>;
  }

  return (
    <main className="container">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center" }}>
        <h1>{status.family.name} - 第{status.family.generation}代</h1>
        <Link href="/" className="small">返回首页</Link>
      </div>

      <section className="card">
        <div className="grid grid-3">
          <div><strong>回合</strong><div>{status.family.turn}</div></div>
          <div><strong>资金</strong><div>{status.resources.family_money}</div></div>
          <div><strong>动作</strong><div>{status.resources.actions_used}/{status.resources.action_quota}</div></div>
          <div><strong>家长精力</strong><div>{status.resources.parent_energy}</div></div>
          <div><strong>孩子精力</strong><div>{status.resources.child_energy}</div></div>
          <div><strong>孩子年龄(月)</strong><div>{status.child.age_month}</div></div>
        </div>
      </section>

      <section className="card" style={{ marginTop: 12 }}>
        <h3>孩子属性：{status.child.name}</h3>
        <div className="grid grid-3">
          <div>智力 {status.child.intelligence}</div>
          <div>自律 {status.child.discipline}</div>
          <div>健康 {status.child.health}</div>
          <div>压力 {status.child.stress}</div>
          <div>自尊 {status.child.self_esteem}</div>
          <div>叛逆 {status.child.rebellion}</div>
          <div>学业 {status.child.study_score}</div>
        </div>
        <p className="small">建议动作：{status.suggested_actions.join(" / ")}</p>
      </section>

      <section className="card" style={{ marginTop: 12 }}>
        <h3>动作面板</h3>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          {actions.map((a) => (
            <button key={a} disabled={busy} onClick={() => doAction(a)}>{a}</button>
          ))}
          <button className="secondary" disabled={busy} onClick={nextTurn}>next-turn</button>
          <button className="secondary" disabled={busy} onClick={nextGeneration}>next-generation</button>
        </div>
        {message ? <p style={{ color: "#b94a48" }}>{message}</p> : null}
      </section>

      <section className="card" style={{ marginTop: 12 }}>
        <h3>操作日志</h3>
        {history.length === 0 ? <p className="small">暂无日志</p> : null}
        {history.map((log, idx) => (
          <div key={`${log.ts}-${idx}`} className="log-item">
            <div style={{ display: "flex", gap: 8 }}>
              <span className="pill">{log.type}</span>
              <strong>{log.title}</strong>
            </div>
            <div className="small">{new Date(log.ts).toLocaleString()}</div>
            <div>{log.detail}</div>
          </div>
        ))}
      </section>
    </main>
  );
}
