"use client";

export default function ClerkMascot({
  mood,
  line
}: {
  mood: "normal" | "warn" | "tired" | "good";
  line: string;
}) {
  return (
    <section className="card clerk-card">
      <div className={`clerk ${mood}`}>
        <div className="clerk-head">
          <span className="eye left" />
          <span className="eye right" />
          <span className="mouth" />
        </div>
        <div className="clerk-body" />
      </div>
      <div>
        <h3 style={{ marginTop: 0, marginBottom: 6 }}>店员小脉</h3>
        <p className="small" style={{ margin: 0 }}>{line}</p>
      </div>
    </section>
  );
}
