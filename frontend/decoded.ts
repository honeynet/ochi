export type DecodedFrame = Record<string, unknown>;

const SHARED_COLUMNS = ['direction', 'command', 'path', 'status', 'truncated'] as const;

const HANDLER_COLUMNS: Record<string, readonly string[]> = {
    smb: ['setup', 'nt_status', 'account', 'native_os', 'native_lanman'],
    http: ['host', 'user_agent', 'status', 'session_id'],
    rdp: ['command', 'cookie', 'protocols'],
    udp: ['payload_hash', 'truncated'],
    tcp: ['payload_hash', 'truncated'],
};

export function isFrameArray(decoded: unknown): decoded is DecodedFrame[] {
    return (
        Array.isArray(decoded) &&
        decoded.every(
            (frame) => frame !== null && typeof frame === 'object' && !Array.isArray(frame),
        )
    );
}

function hasOwn(frame: DecodedFrame, key: string): boolean {
    return Object.prototype.hasOwnProperty.call(frame, key);
}

export function frameTableColumns(frames: DecodedFrame[], handler?: string): string[] {
    const extras = HANDLER_COLUMNS[handler ?? ''] ?? [];
    const ordered = [...SHARED_COLUMNS, ...extras];
    const seen = new Set<string>();
    const columns: string[] = [];
    for (const key of ordered) {
        if (seen.has(key)) {
            continue;
        }
        if (frames.some((frame) => hasOwn(frame, key))) {
            seen.add(key);
            columns.push(key);
        }
    }
    return columns;
}

export function formatFrameValue(value: unknown): string {
    if (value === undefined || value === null) {
        return '';
    }
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
        return String(value);
    }
    if (Array.isArray(value)) {
        if (value.every((item) => typeof item === 'number')) {
            const hex = (value as number[]).map((n) => n.toString(16).padStart(2, '0')).join('');
            return hex ? `0x${hex}` : '';
        }
        return value.map((item) => formatFrameValue(item)).join(', ');
    }
    if (typeof value === 'object') {
        return JSON.stringify(value);
    }
    return String(value);
}

export function formatFrameHeader(header: unknown): string {
    if (header === undefined || header === null) {
        return '';
    }
    if (typeof header !== 'object' || Array.isArray(header)) {
        return formatFrameValue(header);
    }
    const obj = header as Record<string, unknown>;
    const parts: string[] = [];
    for (const key of ['tid', 'uid', 'mid', 'pid', 'flags2', 'Flags2']) {
        if (obj[key] === undefined) {
            continue;
        }
        const label = key.toLowerCase();
        if (!parts.some((part) => part.startsWith(`${label}=`))) {
            parts.push(`${label}=${formatFrameValue(obj[key])}`);
        }
    }
    return parts.length > 0 ? parts.join(' ') : formatFrameValue(header);
}

export function framePayload(frame: DecodedFrame): string | undefined {
    const payload = frame.payload;
    return typeof payload === 'string' && payload.length > 0 ? payload : undefined;
}
