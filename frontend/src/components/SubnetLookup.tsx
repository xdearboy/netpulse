import { useState, useCallback } from 'react';
import { SubnetInfo } from '../types';
import { useAPI } from '../hooks/useAPI';

export const SubnetLookup = () => {
    const [cidr, setCidr] = useState('');
    const [result, setResult] = useState<SubnetInfo | null>(null);
    const { request, loading, error } = useAPI();

    const lookup = useCallback(async () => {
        if (!cidr.trim()) return;
        const data = await request<SubnetInfo>(`/api/v1/subnet/${cidr}`);
        if (data) setResult(data);
    }, [cidr, request]);

    return (
        <div className="space-y-5">
            <div className="flex gap-2">
                <label htmlFor="subnet-input" className="sr-only">CIDR notation</label>
                <input
                    id="subnet-input"
                    value={cidr}
                    onChange={(e) => setCidr(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && lookup()}
                    placeholder="192.168.0.0/24"
                    disabled={loading}
                    autoComplete="off"
                    spellCheck={false}
                    className="flex-1 font-mono bg-white/[0.04] border border-white/[0.08] h-11 px-4 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/20 transition-colors"
                />
                <button
                    onClick={lookup}
                    disabled={loading || !cidr.trim()}
                    aria-label="Lookup subnet"
                    className="bg-white text-black h-11 px-6 text-sm font-medium hover:bg-white/80 active:scale-[0.98] transition-all shrink-0 disabled:opacity-20"
                >
                    {loading ? '…' : 'Lookup →'}
                </button>
            </div>

            {error && (
                <p className="text-sm font-mono text-white/30" role="alert">{error.message}</p>
            )}

            {result && (
                <dl className="divide-y divide-white/[0.06]">
                    {([
                        ['CIDR', result.cidr],
                        ['Network', result.network],
                        ['Netmask', result.netmask],
                        ['Broadcast', result.broadcast],
                        ['IPs', result.ip_count.toLocaleString()],
                        ['Org', result.organization],
                        ['ASN', result.asn ? `AS${result.asn}` : undefined],
                        ['Country', result.country],
                    ] as const).filter(([, v]) => v != null && v !== '').map(([label, value]) => (
                        <div key={label} className="flex items-baseline justify-between py-2.5">
                            <dt className="text-[11px] text-white/25 uppercase tracking-wider">{label}</dt>
                            <dd className="text-sm font-mono text-right tabular-nums">{value}</dd>
                        </div>
                    ))}
                </dl>
            )}
        </div>
    );
};
