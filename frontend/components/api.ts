export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://127.0.0.1:8080";

async function parse<T>(res: Response): Promise<T> {
  const data = (await res.json()) as T & { error?: { code: string; message: string } };
  if (!res.ok) throw new Error(data.error?.message || "请求失败");
  return data as T;
}

export async function registerGame(payload: { agent_id: string; shop_name: string }) {
  const res = await fetch(`${API_BASE}/api/game/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    cache: "no-store"
  });
  return parse<{ game_id: string }>(res);
}

export async function fetchLeaderboard() {
  const res = await fetch(`${API_BASE}/api/leaderboard`, { cache: "no-store" });
  return parse<{ total: number; items: Array<{ game_id: string; shop_name: string; score: number; gold: number; turn: number }> }>(res);
}

export async function fetchConfig() {
  const res = await fetch(`${API_BASE}/api/game/config`, { cache: "no-store" });
  return parse<any>(res);
}
