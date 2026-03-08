"use client";
import Link from "next/link";

export default function Nav() {
  return (
    <nav style={{ display: "flex", gap: 8, marginBottom: 16 }}>
      <Link href="/"><span className="pill">首页</span></Link>
      <Link href="/skill"><span className="pill">Agent Skill</span></Link>
    </nav>
  );
}
