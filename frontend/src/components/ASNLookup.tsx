import { useState, useCallback } from 'react';
import { ASNInfo } from '../types';
import { useAPI } from '../hooks/useAPI';

export const ASNLookup = () => {
    const [asn, setAsn] = useState('');
    const [result, setResult] = useState<ASNInfo | null>(null);
    const { request, loading, error } = useAPI();

    const lookup = useCallback(async () => {
        const num = parseInt(asn);
        if (isNaN(num)) return;
        const data = await request<ASNInfo>(`/api/v1/asn/${num}`);
        if (data) setResult(data);
    }, [asn, request]);

    return (
        <div className="space-y-5">
            <div className="flex gap-2">
                <label htmlFor="asn-input" className="sr-only">ASN number</label>
                <input
                    id="asn-input"
                    value={asn}
                    onChange={(e) => setAsn(e.target.value.replace(/\D/g, ''))}
                    onKeyDown={(e) => e.key === 'Enter' && lookup()}
                    placeholder="15169"
                    disabled={loading}
                    autoComplete="off"
                    inputMode="numeric"
                    className="flex-1 font-mono bg-white/[0.04] border border-white/[0.08] h-11 px-4 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/20 transition-colors"
                />
                <button
                    onClick={lookup}
                    disabled={loading || !asn.trim()}
                    aria-label="Lookup ASN"
                    className="bg-white text-black h-11 px-6 text-sm font-medium hover:bg-white/80 active:scale-[0.98] transition-all shrink-0 disabled:opacity-20"
                >
                    {loading ? '…' : 'Lookup →'}
                </button>
            </div>

            {error && (
                <p className="text-sm font-mono text-white/30" role="alert">{error.message}</p>
            )}

            {result && (
                <div className="space-y-6">
                    <dl className="divide-y divide-white/[0.06]">
                        {([
                            ['ASN', `AS${result.asn}`],
                            ['Name', result.as_name],
                            ['Org', result.organization],
                            ['Country', result.country],
                            ['Registry', result.registry],
                            ['Status', result.status],
                            ['Announced', result.announced != null ? (result.announced ? 'Yes' : 'No') : undefined],
                            ['Prefixes', result.prefix_count?.toLocaleString()],
                            ['Peers', result.peer_count?.toLocaleString()],
                        ] as const).filter(([, v]) => v != null && v !== '').map(([label, value]) => (
                            <div key={label} className="flex items-baseline justify-between py-2.5">
                                <dt className="text-[11px] text-white/25 uppercase tracking-wider">{label}</dt>
                                <dd className="text-sm font-mono text-right tabular-nums">{value}</dd>
                            </div>
                        ))}
                    </dl>

                    {result.prefixes && result.prefixes.length > 0 && (
                        <div>
                            <p className="text-[11px] text-white/15 uppercase tracking-[0.15em] mb-2">Prefixes</p>
                            <div className="flex flex-wrap gap-1.5">
                                {result.prefixes.slice(0, 12).map((p) => (
                                    <span key={p} className="text-xs font-mono bg-white/[0.04] border border-white/[0.08] px-2 py-1 text-white/40">{p}</span>
                                ))}
                                {result.prefixes.length > 12 && (
                                    <span className="text-xs text-white/15 self-center">+{result.prefixes.length - 12}</span>
                                )}
                            </div>
                        </div>
                    )}

                    {result.peers && result.peers.length > 0 && (
                        <div>
                            <p className="text-[11px] text-white/15 uppercase tracking-[0.15em] mb-2">Peers</p>
                            <div className="flex flex-wrap gap-1.5">
                                {result.peers.slice(0, 12).map((p) => (
                                    <span key={p} className="text-xs font-mono bg-white/[0.04] border border-white/[0.08] px-2 py-1 text-white/40">{p}</span>
                                ))}
                                {result.peers.length > 12 && (
                                    <span className="text-xs text-white/15 self-center">+{result.peers.length - 12}</span>
                                )}
                            </div>
                        </div>
                    )}

                    {result.upstreams && result.upstreams.length > 0 && (
                        <div>
                            <p className="text-[11px] text-white/15 uppercase tracking-[0.15em] mb-2">Upstreams</p>
                            <div className="flex flex-wrap gap-1.5">
                                {result.upstreams.slice(0, 12).map((u) => (
                                    <span key={u} className="text-xs font-mono bg-white/[0.04] border border-white/[0.08] px-2 py-1 text-white/40">{u}</span>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};
