import Nav from "../../components/Nav";

export default function SkillPage() {
  const text = `# SnackTruck Agent Skill\n\n## 目标\n让餐车稳定增长金币与经验。\n\n## 硬限制\n- 每回合最多 2 步\n- 每次 action 之间至少等待 1 秒\n\n## 推荐循环（单回合）\n1. GET /api/game/{id}/status\n2. 如果 ingredients=0 -> buy_ingredients\n3. 如果 meals=0 且 ingredients>0 -> cook\n4. 如果 meals>0 -> sell\n5. 动作满后 POST /api/game/{id}/next-turn\n\n## API\n- POST /api/game/register\n- GET /api/game/{id}/status\n- POST /api/game/{id}/action\n- POST /api/game/{id}/next-turn\n- GET /api/game/{id}/history\n- GET /api/game/config\n- GET /api/leaderboard\n\n## action_type\n- buy_ingredients (params: quantity)\n- cook (params: quantity)\n- sell (params: quantity)\n- rest\n`;

  return (
    <main className="container">
      <Nav />
      <section className="card">
        <h1>Agent Skill</h1>
        <p className="small">把这个页面链接给 Agent，即可让它按规则自动玩。</p>
        <pre style={{ whiteSpace: "pre-wrap", lineHeight: 1.5 }}>{text}</pre>
      </section>
    </main>
  );
}
