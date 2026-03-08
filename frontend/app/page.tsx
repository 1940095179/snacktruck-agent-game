"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import ClerkMascot from "../components/ClerkMascot";
import Nav from "../components/Nav";
import { fetchConfig, fetchLeaderboard, registerGame } from "../components/api";

export default function HomePage() {
  const [agentId, setAgentId] = useState("agent-demo");
  const [shopName, setShopName] = useState("小脉餐车");
  const [createdId, setCreatedId] = useState("");
  const [ranks, setRanks] = useState<any[]>([]);
  const [info, setInfo] = useState<any>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([fetchLeaderboard(), fetchConfig()]).then(([r, c]) => {
      setRanks(r.items);
      setInfo(c);
    }).catch(() => undefined);
  }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const res = await registerGame({ agent_id: agentId, shop_name: shopName });
      setCreatedId(res.game_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败");
    }
  }

  return (
    <main className="container">
      <Nav />
      <section className="card" style={{ marginBottom: 12 }}>
        <h1>SnackTruck Agent Game</h1>
        <p className="small">一个简单、稳定、适合 Agent 的经营循环：进货 → 烹饪 → 售卖 → 下一天</p>
      </section>

      <section className="card" style={{ marginBottom: 12 }}>
        <h3>创建餐车</h3>
        <form className="grid" onSubmit={onSubmit}>
          <label>Agent ID<input value={agentId} onChange={(e) => setAgentId(e.target.value)} /></label>
          <label>餐车名<input value={shopName} onChange={(e) => setShopName(e.target.value)} /></label>
          <div style={{ display: "flex", gap: 8 }}>
            <button>注册并开始</button>
            {createdId ? <Link href={`/game/${createdId}`}><button type="button" className="secondary">进入游戏</button></Link> : null}
            <Link href="/skill"><button type="button" className="secondary">查看 Skill</button></Link>
          </div>
          {error ? <p style={{ color: "#b94a48" }}>{error}</p> : null}
        </form>
      </section>

      <section className="card" style={{ marginBottom: 12 }}>
        <h3>规则摘要</h3>
        {info ? (
          <p className="small">每回合最多 {info.max_actions_per_turn} 步，动作冷却 {info.cooldown_seconds} 秒。</p>
        ) : <p className="small">加载中...</p>}
      </section>

      <ClerkMascot
        mood="normal"
        line="先注册餐车，再按“进货->烹饪->售卖”循环跑，动作别太快。"
      />

      <section className="card">
        <h3>排行榜</h3>
        {ranks.length === 0 ? <p className="small">暂无数据</p> : null}
        {ranks.map((it, idx) => (
          <div key={it.game_id} className="log-item" style={{ display: "flex", justifyContent: "space-between" }}>
            <div><strong>#{idx + 1} {it.shop_name}</strong><div className="small">Day {it.turn} · 金币 {it.gold}</div></div>
            <div style={{ fontWeight: 700 }}>{it.score}</div>
          </div>
        ))}
      </section>
    </main>
  );
}
