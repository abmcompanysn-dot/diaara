'use client';

import { useMemo, useState } from 'react';

interface Point {
  day: string;
  sales: number;
  revenue_cfa: number;
}

interface SalesChartProps {
  data: Point[];
  height?: number;
}

// Graphique en barres léger (aucune dépendance externe) : volume de ventes
// par jour, avec l'infobulle au survol affichant le détail du jour.
export function SalesChart({ data, height = 220 }: SalesChartProps) {
  const [hover, setHover] = useState<number | null>(null);

  const max = useMemo(() => Math.max(1, ...data.map((d) => d.sales)), [data]);
  const width = Math.max(data.length * 28, 280);
  const barWidth = Math.max((width / Math.max(data.length, 1)) * 0.6, 3);
  const gap = width / Math.max(data.length, 1);

  if (data.length === 0) {
    return (
      <div className="h-[220px] flex items-center justify-center text-sm text-green-900/40 font-mono">
        Pas encore de données sur cette période
      </div>
    );
  }

  const active = hover !== null ? data[hover] : null;

  return (
    <div className="relative">
      {active && (
        <div className="absolute -top-1 right-0 text-right">
          <p className="text-xs font-mono text-green-900/50">
            {new Date(active.day).toLocaleDateString('fr-FR', { day: '2-digit', month: 'short' })}
          </p>
          <p className="text-sm font-bold text-green-950">
            {active.sales} vente{active.sales > 1 ? 's' : ''} &middot;{' '}
            {active.revenue_cfa.toLocaleString('fr-FR')} FCFA
          </p>
        </div>
      )}
      <div className="overflow-x-auto no-scrollbar">
        <svg
          width={width}
          height={height}
          className="block"
          role="img"
          aria-label="Évolution des ventes"
        >
          {data.map((d, i) => {
            const barHeight = Math.max((d.sales / max) * (height - 24), d.sales > 0 ? 3 : 0);
            const x = i * gap + (gap - barWidth) / 2;
            const y = height - barHeight - 20;
            const isHover = hover === i;
            return (
              <g key={d.day}>
                <rect
                  x={x}
                  y={y}
                  width={barWidth}
                  height={barHeight}
                  rx={3}
                  className={isHover ? 'fill-green-600' : 'fill-lime'}
                  onMouseEnter={() => setHover(i)}
                  onMouseLeave={() => setHover(null)}
                />
                <rect
                  x={x - gap * 0.15}
                  y={0}
                  width={gap * 1.3}
                  height={height}
                  fill="transparent"
                  onMouseEnter={() => setHover(i)}
                  onMouseLeave={() => setHover(null)}
                />
                {(i === 0 || i === data.length - 1 || i % Math.ceil(data.length / 6) === 0) && (
                  <text
                    x={x + barWidth / 2}
                    y={height - 4}
                    textAnchor="middle"
                    className="fill-green-900/40 font-mono"
                    fontSize={10}
                  >
                    {new Date(d.day).toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })}
                  </text>
                )}
              </g>
            );
          })}
        </svg>
      </div>
    </div>
  );
}
