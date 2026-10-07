export type DecodedFrame = Record<string, unknown>;

const SHARED_COLUMNS = ['direction', 'command', 'path', 'status', 'truncated'] as const;

// Columns that are never shown in the frame table; payload is shown as an
// expandable hexdump instead.
const HIDDEN_COLUMNS = new Set(['payload']);

// Preferred column order per handler. Keys present in the frames but not listed
// here are still shown, after these, in the order they first appear.
const HANDLER_COLUMNS: Record<string, readonly string[]> = {
    smb: ['header', 'setup', 'nt_status', 'account', 'native_os', 'native_lanman'],
    http: ['host', 'query', 'user_agent', 'dest_port', 'src_port', 'session_id'],
    rdp: ['cookie', 'protocols', 'ntlm_domain', 'ntlm_user', 'ntlm_workstation'],
    udp: ['payload_hash'],
    tcp: ['payload_hash'],
};

// Fixed-size byte arrays, which Go marshals as JSON number arrays. Every other
// byte slice is marshalled as a base64 string, so other number arrays are
// lists of IDs (cipher suites, etypes, encodings) and must not be hex-joined.
const BYTE_ARRAY_KEYS = new Set(['protocol_identifier', 'reserved', 'info_hash', 'peer_id']);

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
    const seen = new Set<string>();
    const columns: string[] = [];
    const add = (key: string) => {
        if (seen.has(key) || HIDDEN_COLUMNS.has(key)) {
            return;
        }
        seen.add(key);
        columns.push(key);
    };
    for (const key of [...SHARED_COLUMNS, ...extras]) {
        if (frames.some((frame) => hasOwn(frame, key))) {
            add(key);
        }
    }
    for (const frame of frames) {
        for (const key of Object.keys(frame)) {
            add(key);
        }
    }
    return columns;
}

function isByteArray(value: unknown[]): value is number[] {
    return value.every((n) => typeof n === 'number' && Number.isInteger(n) && n >= 0 && n <= 255);
}

export function formatFrameValue(value: unknown, key?: string): string {
    if (value === undefined || value === null) {
        return '';
    }
    if (key === 'header') {
        return formatFrameHeader(value);
    }
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
        return String(value);
    }
    if (Array.isArray(value)) {
        if (key !== undefined && BYTE_ARRAY_KEYS.has(key) && isByteArray(value)) {
            const hex = value.map((n) => n.toString(16).padStart(2, '0')).join('');
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
    return parts.length > 0 ? parts.join(' ') : JSON.stringify(header);
}

export function framePayload(frame: DecodedFrame): string | undefined {
    const payload = frame.payload;
    return typeof payload === 'string' && payload.length > 0 ? payload : undefined;
}

// Formats the frame count as received/sent, where received frames are the
// client's reads and sent frames are the handler's writes. Falls back to the
// sensor's total frame count when the decoded frames carry no direction.
export function formatFrameCount(decoded: unknown, frameCount?: number): string {
    if (isFrameArray(decoded) && decoded.some((frame) => typeof frame.direction === 'string')) {
        const sent = decoded.filter((frame) => frame.direction === 'write').length;
        return `${decoded.length - sent}/${sent}`;
    }
    return frameCount !== undefined ? String(frameCount) : '';
}
