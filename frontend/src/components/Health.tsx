import { useEffect, useState, useRef } from 'react';
import { HealthResponse } from '../types';
import { useAPI } from '../hooks/useAPI';

export const Health = () => {
    const [health, setHealth] = useState<HealthResponse | null>(null);
    const { request } = useAPI();
    const mountedRef = useRef(true);

    useEffect(() => {
        mountedRef.current = true;
        const load = async () => {
            const data = await request<HealthResponse>('/health');
            if (data && mountedRef.current) setHealth(data);
        };
        load();
        const id = setInterval(load, 8000);
        return () => { mountedRef.current = false; clearInterval(id); };
    }, [request]);

    if (!health) {
        return (
            <div className="space-y-2.5" aria-label="Loading health status…">
                {[1, 2, 3, 4].map((i) => (
                    <div key={i} className="h-4 bg-white/[0.04] animate-pulse" />
                ))}
            </div>
        );
    }

    const ok = health.status === 'ok';

    return (
        <div className="space-y-4" role="status" aria-live="polite">
            <div className="flex items-center gap-2">
                <span
                    className="w-1.5 h-1.5 rounded-full"
                    style={{ background: ok ? 'var(--color-ok)' : 'var(--color-warn)' }}
                    aria-hidden="true"
                />
                <span className="text-sm font-mono text-white/40">{health.status}</span>
            </div>

            <div className="space-y-2">
                {([
                    ['Sources', `${health.summary.healthy_sources}/${health.summary.total_sources}`],
                    ['Uptime', health.summary.uptime],
                    ['Requests', health.summary.total_requests.toLocaleString()],
                    ['Goroutines', String(health.summary.go_routines)],
                ] as const).map(([label, value]) => (
                    <div key={label} className="flex items-center justify-between">
                        <span className="text-[11px] text-white/20 uppercase tracking-wider">{label}</span>
                        <span className="text-xs font-mono text-white/40 tabular-nums">{value}</span>
                    </div>
                ))}
            </div>

            <div className="pt-2 border-t border-white/[0.06]">
                <p className="text-[10px] text-white/12 uppercase tracking-[0.15em] mb-2">Sources</p>
                <div className="space-y-1">
                    {Object.entries(health.sources).map(([name, s]) => (
                        <div key={name} className="flex items-center justify-between text-[11px] font-mono">
                            <span className="text-white/25">{name}</span>
                            <div className="flex items-center gap-2">
                                {s.latency && <span className="text-white/12 tabular-nums">{s.latency}</span>}
                                <span
                                    className="tabular-nums"
                                    style={{ color: s.status === 'ok' ? 'var(--color-ok)' : 'var(--color-err)', opacity: 0.5 }}
                                >
                                    {s.status}
                                </span>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
};
