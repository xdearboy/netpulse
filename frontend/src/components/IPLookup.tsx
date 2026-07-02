import { useState, useCallback } from 'react';
import { IPResult } from '../types';
import { useAPI } from '../hooks/useAPI';

export const IPLookup = () => {
    const [ip, setIp] = useState('');
    const [result, setResult] = useState<IPResult | null>(null);
    const [copied, setCopied] = useState(false);
    const { request, loading, error } = useAPI();

    const lookup = useCallback(async () => {
        const target = ip.trim() || 'me';
        const data = await request<IPResult>(`/api/v1/ip/${target}`);
        if (data) setResult(data);
    }, [ip, request]);

    const copyJSON = useCallback(() => {
        if (!result) return;
        navigator.clipboard.writeText(JSON.stringify(result, null, 2));
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
    }, [result]);

    const rows: [string, string | number | undefined][] = [
        ['IP', result?.ip_address],
        ['Country', result?.country_name],
        ['City', result?.city],
        ['Region', result?.region],
        ['ISP', result?.isp],
        ['Org', result?.organization],
        ['ASN', result?.asn ? `AS${result.asn}` : undefined],
        ['AS Name', result?.as_name],
        ['Coords', result ? `${result.latitude.toFixed(4)}, ${result.longitude.toFixed(4)}` : undefined],
        ['Timezone', result?.timezone],
        ['ZIP', result?.zip],
    ];

    return (
        <div className="space-y-5">
            <div className="flex gap-2">
                <label htmlFor="ip-input" className="sr-only">IP address</label>
                <input
                    id="ip-input"
                    value={ip}
                    onChange={(e) => setIp(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && lookup()}
                    placeholder="8.8.8.8 or leave empty for yours"
                    disabled={loading}
                    autoComplete="off"
                    spellCheck={false}
                    className="flex-1 font-mono bg-white/[0.04] border border-white/[0.08] h-11 px-4 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/20 transition-colors"
                />
                <button
                    onClick={lookup}
                    disabled={loading}
                    aria-label="Lookup IP address"
                    className="bg-white text-black h-11 px-6 text-sm font-medium hover:bg-white/80 active:scale-[0.98] transition-all shrink-0"
                >
                    {loading ? '…' : 'Lookup →'}
                </button>
            </div>

            {error && (
                <p className="text-sm font-mono text-white/30" role="alert">
                    {error.status === 429
                        ? `Rate limited — retry in ${error.retryAfter ?? '?'}s`
                        : error.message}
                </p>
            )}

            {result && (
                <div className="space-y-4">
                    <div className="flex items-center justify-between">
                        <span className="text-[11px] font-mono text-white/15 uppercase tracking-widest">
                            {result.sources_count} sources · {result.query_time}
                        </span>
                        <button
                            onClick={copyJSON}
                            className="text-[12px] font-mono text-white/20 hover:text-white/50 transition-colors"
                            aria-label="Copy result as JSON"
                        >
                            {copied ? 'copied' : '{ }'}
                        </button>
                    </div>
                    <dl className="divide-y divide-white/[0.06]">
                        {rows.map(([label, value]) => (
                            <div key={label} className="flex items-baseline justify-between py-2.5">
                                <dt className="text-[11px] text-white/25 uppercase tracking-wider">{label}</dt>
                                <dd className="text-sm font-mono text-right break-all max-w-[60%] tabular-nums">
                                    {value ?? '—'}
                                </dd>
                            </div>
                        ))}
                    </dl>
                    {result.sources_failed && result.sources_failed.length > 0 && (
                        <p className="text-[11px] font-mono text-white/15">
                            failed: {result.sources_failed.join(', ')}
                        </p>
                    )}
                </div>
            )}
        </div>
    );
};
