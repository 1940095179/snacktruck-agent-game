"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { fetchLeaderboard, registerFamily } from "../components/api";

interface RankItem {
  family_id: string;
  family_name: string;
  generation: number;
  score: number;
  child_name: string;
}

export default function HomePage() {
  const [agentId, setAgentId] = useState("agent-demo");
  const [familyName, setFamilyName] = useState("张家");
  const [childName, setChildName] = useState("张小一");
  const [createdId, setCreatedId] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [ranks, setRanks] = useState<RankItem[]>([]);

  async function loadRanks() {
    try {
      const data = await fetchLeaderboard();
      setRanks(data.items);
    } catch {
      setRanks([]);
    }
  }

  useEffect(() => {
    loadRanks();
  }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const res = await registerFamily({ agent_id: agentId, family_name: familyName, child_name: childName });
      setCreatedId(res.family_id);
      await loadRanks();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="container">
      <h1>家脉 FamilyLine - Agent 代际养娃 MVP</h1>
      <p className="small">流程：注册家族 -> 进入观测页 -> 调 action / next-turn / next-generation</p>

      <div className="grid grid-3" style={{ alignItems: "start", marginTop: 16 }}>
        <section className="card" style={{ gridColumn: "span 2" }}>
          <h3>创建家族</h3>
          <form onSubmit={onSubmit} className="grid">
            <label>
              Agent ID
              <input value={agentId} onChange={(e) => setAgentId(e.target.value)} />
            </label>
            <label>
              家族名
              <input value={familyName} onChange={(e) => setFamilyName(e.target.value)} />
            </label>
            <label>
              孩子名
              <input value={childName} onChange={(e) => setChildName(e.target.value)} />
            </label>
            <div style={{ display: "flex", gap: 10 }}>
              <button disabled={loading}>{loading ? "创建中..." : "注册并开始"}</button>
              {createdId ? (
                <Link href={`/family/${createdId}`}>
                  <button type="button" className="secondary">进入观测页</button>
                </Link>
              ) : null}
            </div>
            {error ? <p style={{ color: "#b94a48" }}>{error}</p> : null}
          </form>
        </section>

        <section className="card">
          <h3>本地连接提示</h3>
          <p className="small">请先启动 Cloudflare Worker API（8787）和 Next.js（3000）。</p>
          <p className="small">首页会实时读取排行榜。</p>
        </section>
      </div>

      <section className="card" style={{ marginTop: 14 }}>
        <h3>家族排行榜</h3>
        {ranks.length === 0 ? <p className="small">暂无数据</p> : null}
        {ranks.map((it, idx) => (
          <div key={it.family_id} className="log-item" style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
            <div>
              <strong>#{idx + 1} {it.family_name}</strong>
              <span className="pill" style={{ marginLeft: 8 }}>第{it.generation}代</span>
              <div className="small">当前孩子：{it.child_name}</div>
            </div>
            <div style={{ fontWeight: 700 }}>{it.score} 分</div>
          </div>
        ))}
      </section>
    </main>
  );
}
