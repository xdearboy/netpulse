import { useState } from 'react';
import { IPLookup } from './components/IPLookup';
import { ASNLookup } from './components/ASNLookup';
import { SubnetLookup } from './components/SubnetLookup';
import { Health } from './components/Health';

type Tab = 'ip' | 'asn' | 'subnet';

const tabs: { id: Tab; label: string }[] = [
    { id: 'ip', label: 'IP' },
    { id: 'asn', label: 'ASN' },
    { id: 'subnet', label: 'Subnet' },
];

const examples = [
    { label: 'IP', cmd: 'curl netpulse.digital/api/v1/ip/8.8.8.8' },
    { label: 'ASN', cmd: 'curl netpulse.digital/api/v1/asn/15169' },
    { label: 'Subnet', cmd: 'curl netpulse.digital/api/v1/subnet/192.168.0.0/24' },
    { label: 'Health', cmd: 'curl netpulse.digital/health' },
];

function App() {
    const [activeTab, setActiveTab] = useState<Tab>('ip');

    return (
        <div className="min-h-screen bg-black text-white">
            {/* nav */}
            <header className="border-b border-white/[0.06]">
                <nav className="max-w-6xl mx-auto flex items-center justify-between px-6 h-14">
                    <span className="text-sm font-semibold tracking-tight">netpulse</span>
                    <div className="flex items-center gap-6 text-[13px] text-white/30">
                        <a href="https://github.com/xdearboy/netpulse" target="_blank" rel="noopener noreferrer"
                           className="hover:text-white/60 transition-colors">GitHub</a>
                        <a href="/docs" className="hover:text-white/60 transition-colors">Docs</a>
                    </div>
                </nav>
            </header>

            {/* hero */}
            <section className="max-w-6xl mx-auto px-6 pt-20 pb-16 md:pt-28 md:pb-20">
                <h1 className="text-4xl md:text-5xl font-semibold tracking-[-0.03em] leading-[1.1] max-w-lg">
                    IP geolocation<br />
                    <span className="text-white/20">for everyone.</span>
                </h1>
                <p className="mt-5 text-[15px] text-white/30 leading-relaxed max-w-md">
                    Seven sources, parallel queries, consensus voting.
                    Free, open-source, MIT licensed.
                </p>
                <a href="/docs" className="inline-block mt-6 text-[14px] text-white/40 hover:text-white/70 transition-colors">
                    Documentation →
                </a>
            </section>

            {/* api */}
            <section className="max-w-6xl mx-auto px-6 pb-16 md:pb-24">
                <div className="grid grid-cols-1 md:grid-cols-[1fr_340px] gap-12 md:gap-16">
                    {/* left: lookup panel */}
                    <div>
                        <div className="flex items-center gap-6 mb-6 border-b border-white/[0.06]" role="tablist" aria-label="Lookup type">
                            {tabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    role="tab"
                                    aria-selected={activeTab === tab.id}
                                    aria-controls={`panel-${tab.id}`}
                                    id={`tab-${tab.id}`}
                                    onClick={() => setActiveTab(tab.id)}
                                    className={`pb-3 text-[13px] font-medium transition-colors border-b-2 -mb-px ${
                                        activeTab === tab.id
                                            ? 'text-white border-white'
                                            : 'text-white/25 border-transparent hover:text-white/40'
                                    }`}
                                >
                                    {tab.label}
                                </button>
                            ))}
                        </div>
                        <div role="tabpanel" id={`panel-${activeTab}`} aria-labelledby={`tab-${activeTab}`}>
                            {activeTab === 'ip' && <IPLookup />}
                            {activeTab === 'asn' && <ASNLookup />}
                            {activeTab === 'subnet' && <SubnetLookup />}
                        </div>
                    </div>

                    {/* right: sidebar */}
                    <div className="space-y-10">
                        {/* status */}
                        <div>
                            <h2 className="text-[11px] text-white/20 uppercase tracking-[0.15em] mb-4">Status</h2>
                            <Health />
                        </div>

                        {/* quick start */}
                        <div>
                            <h2 className="text-[11px] text-white/20 uppercase tracking-[0.15em] mb-4">Quick start</h2>
                            <div className="space-y-2">
                                {examples.map((ex) => (
                                    <div key={ex.label} className="flex items-baseline gap-3">
                                        <span className="text-[10px] text-white/12 w-12 shrink-0 uppercase tracking-wider">{ex.label}</span>
                                        <code className="text-[12px] font-mono text-white/25 break-all">{ex.cmd}</code>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                </div>
            </section>

            {/* stats bar */}
            <section className="border-t border-white/[0.06]">
                <div className="max-w-6xl mx-auto px-6 py-10 grid grid-cols-2 md:grid-cols-4 gap-8">
                    {([
                        ['7', 'Sources', 'free APIs queried in parallel'],
                        ['Vote', 'Method', 'strings by majority, coords by median'],
                        ['MIT', 'License', 'fork it, mirror it, run it yourself'],
                        ['< 10ms', 'Cached', 'in-memory cache with BigCache'],
                    ] as const).map(([big, label, desc]) => (
                        <div key={label}>
                            <p className="text-2xl md:text-3xl font-light tracking-tight text-white/80">{big}</p>
                            <p className="text-[12px] font-medium text-white/40 mt-1">{label}</p>
                            <p className="text-[11px] text-white/15 mt-0.5">{desc}</p>
                        </div>
                    ))}
                </div>
            </section>

            {/* footer */}
            <footer className="border-t border-white/[0.06]">
                <div className="max-w-6xl mx-auto px-6 py-5 flex items-center justify-between text-[12px] text-white/15">
                    <span>Built by xdearboy</span>
                    <a href="https://github.com/xdearboy/netpulse" className="hover:text-white/30 transition-colors">GitHub</a>
                </div>
            </footer>
        </div>
    );
}

export default App;
