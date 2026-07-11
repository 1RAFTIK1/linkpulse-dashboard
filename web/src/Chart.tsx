import { useEffect, useState } from "react";
import type { ClickData } from "./types";

const WINDOW_MIN = 15; // окно графика: последние 15 минут

// Chart — поминутные столбики кликов, чистый SVG без библиотек.
// Тикаем раз в секунду, чтобы окно ползло вперёд даже без новых кликов.
export function Chart({ clicks }: { clicks: ClickData[] }) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(t);
  }, []);

  const minuteMs = 60_000;
  const currentMinute = Math.floor(now / minuteMs);
  const counts = new Array<number>(WINDOW_MIN).fill(0);
  for (const c of clicks) {
    const idx = currentMinute - Math.floor(new Date(c.clicked_at).getTime() / minuteMs);
    if (idx >= 0 && idx < WINDOW_MIN) counts[WINDOW_MIN - 1 - idx]++;
  }
  const max = Math.max(1, ...counts);

  const w = 600;
  const h = 140;
  const barW = w / WINDOW_MIN;

  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="chart" role="img" aria-label="Клики по минутам">
      {counts.map((n, i) => {
        const barH = (n / max) * (h - 24);
        return (
          <g key={i}>
            <rect
              x={i * barW + 3}
              y={h - 16 - barH}
              width={barW - 6}
              height={Math.max(barH, n > 0 ? 3 : 0)}
              rx={3}
              className="chart-bar"
            />
            {n > 0 && (
              <text x={i * barW + barW / 2} y={h - 20 - barH} textAnchor="middle" className="chart-count">
                {n}
              </text>
            )}
          </g>
        );
      })}
      <text x={4} y={h - 4} className="chart-axis">−{WINDOW_MIN} мин</text>
      <text x={w - 4} y={h - 4} textAnchor="end" className="chart-axis">сейчас</text>
    </svg>
  );
}
