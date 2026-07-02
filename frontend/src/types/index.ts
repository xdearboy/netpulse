export interface IPResult {
    ip_address: string;
    type: string;
    country: string;
    country_name: string;
    city: string;
    region: string;
    latitude: number;
    longitude: number;
    isp: string;
    organization: string;
    asn: number;
    as_name: string;
    timezone: string;
    zip: string;
    sources_used: string[];
    sources_failed?: string[];
    sources_count: number;
    query_time: string;
    cached_at: string;
}

export interface ASNInfo {
    asn: number;
    as_name: string;
    organization: string;
    country?: string;
    registry?: string;
    status?: string;
    announced?: boolean;
    prefix_count?: number;
    peer_count?: number;
    prefixes?: string[];
    peers?: string[];
    upstreams?: string[];
    admin_contacts?: string[];
    tech_contacts?: string[];
    block_description?: string;
    registered?: string;
    cached_at?: string;
}

export interface SubnetInfo {
    cidr: string;
    ip_count: number;
    netmask: string;
    network: string;
    broadcast?: string;
    organization?: string;
    asn?: number;
    as_name?: string;
    country?: string;
    cached_at?: string;
}

export interface HealthResponse {
    status: string;
    sources: Record<string, { status: string; latency?: string }>;
    total_time: string;
    summary: {
        total_sources: number;
        healthy_sources: number;
        uptime: string;
        total_requests: number;
        active_requests: number;
        go_routines: number;
    };
}
