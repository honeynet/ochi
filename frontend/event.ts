import { API_ENDPOINT } from './constants';

export interface TLSInfo {
    serverName?: string; // SNI
    alpn?: string[]; // offered protocols
    version?: string;
    cipher?: string; // empty when the handshake failed
    clientHello?: string; // base64 of the raw ClientHello records, capped at 4 KiB
    truncated?: boolean;
}

export interface Event {
    id?: string;
    ownerID?: string;
    payload?: string; // base64 encoded binary payload
    connKey?: number[]; // identifier based on IP and source port
    dstPort: number; // the connection destination port
    rule?: string; // the rule that matched the connection
    handler?: string; // the processing handler
    transport?: string;
    scanner?: string; // name of the scanner if detected
    sensorID: string; // the id of the sensor
    srcHost: string; // the source IP address
    srcPort: string; // the source port
    timestamp: string; // the UTC timestamp of the connection
    decoded?: unknown; // a decoded version of the payload if available
    startedAt?: string;
    durationMs?: number;
    srcPtr?: string;
    dstHost?: string;
    sensorVersion?: string;
    ruleName?: string;
    payloadHash?: string;
    frameCount?: number;
    endReason?: string;
    tls?: TLSInfo; // set when the sensor terminated TLS; payload and decoded are then plaintext
}

export function formatPort(event: Event): string {
    return event.transport ? `${event.transport}/${event.dstPort}` : `:${event.dstPort}`;
}

export function formatDest(event: Event): string {
    if (event.dstHost) {
        return `${event.dstHost} ${formatPort(event)}`;
    }
    return formatPort(event);
}

// Go marshals a zero time.Time despite omitempty.
export function hasTime(value?: string): value is string {
    return !!value && !value.startsWith('0001-01-01');
}

export function formatDuration(ms: number): string {
    if (ms < 1000) {
        return `${ms}ms`;
    }
    if (ms < 60_000) {
        return `${(ms / 1000).toFixed(1)}s`;
    }
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = String(totalSeconds % 60).padStart(2, '0');
    if (hours > 0) {
        return `${hours}h${String(minutes).padStart(2, '0')}m${seconds}s`;
    }
    return `${minutes}m${seconds}s`;
}

export const END_REASONS: Record<string, string> = {
    client_close: 'the client closed the session',
    timeout: 'the session idled out',
    handler_close: 'the handler closed the session',
    read_error: 'reading from the client failed or the input did not parse',
    write_error: 'writing to the client failed',
    max_frames: 'the session hit the frame cap',
    evicted:
        "the handler's session table was full and this least recently active session was flushed early",
};

export function displayRule(event: Event): string | undefined {
    return event.ruleName || event.rule;
}

export interface EventPage {
    events: Event[];
    total: number;
}

export async function getEvents(token: string, limit: number, offset: number): Promise<EventPage> {
    const res = await fetch(`${API_ENDPOINT}/api/events?limit=${limit}&offset=${offset}`, {
        method: 'GET',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
    });

    if (!res.ok) {
        throw new Error('Could not fetch events');
    }
    const total = Number.parseInt(res.headers.get('X-Total-Count') ?? '', 10);
    if (Number.isNaN(total)) {
        throw new Error('Missing event count');
    }
    return { events: (await res.json()) ?? [], total };
}

export async function deleteEvent(id: string, token: string): Promise<void> {
    const res = await fetch(`${API_ENDPOINT}/api/events/${id}`, {
        method: 'DELETE',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
    });

    if (!res.ok) {
        throw new Error('Could not delete an event');
    }
}

export async function deleteEvents(ids: string[], token: string): Promise<void> {
    const res = await fetch(`${API_ENDPOINT}/api/events`, {
        method: 'DELETE',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ ids }),
    });

    if (!res.ok) {
        throw new Error('Could not delete events');
    }
}
