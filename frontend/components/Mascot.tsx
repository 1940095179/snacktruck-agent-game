"use client";

type Mood = "happy" | "tired" | "stressed" | "focused";

export default function Mascot({
  name,
  mood,
  line
}: {
  name: string;
  mood: Mood;
  line: string;
}) {
  return (
    <section className="card mascot-card">
      <div className={`mascot ${mood}`}>
        <div className="mascot-head">
          <span className="eye left" />
          <span className="eye right" />
          <span className="mouth" />
        </div>
        <div className="mascot-body" />
      </div>
      <div>
        <h3 style={{ marginTop: 0 }}>{name}</h3>
        <p className="small" style={{ marginBottom: 0 }}>{line}</p>
      </div>
    </section>
  );
}
