import "./globals.css";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "家脉 FamilyLine",
  description: "面向 Agent 的代际养娃模拟器"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}
