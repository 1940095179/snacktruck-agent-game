const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://127.0.0.1:8080";

async function parse<T>(res: Response): Promise<T> {
  const data = (await res.json()) as T & { error?: { code: string; message: string } };
  if (!res.ok) {
    throw new Error(data.error?.message || "请求失败");
  }
  return data as T;
}

export async function registerFamily(payload: {
  agent_id: string;
  family_name: string;
  child_name: string;
}) {
  const res = await fetch(`${API_BASE}/api/family/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    cache: "no-store"
  });
  return parse<{ family_id: string }>(res);
}

export async function fetchLeaderboard() {
  const res = await fetch(`${API_BASE}/api/leaderboard`, { cache: "no-store" });
  return parse<{
    total: number;
    items: Array<{ family_id: string; family_name: string; generation: number; score: number; child_name: string }>;
  }>(res);
}

export { API_BASE };
