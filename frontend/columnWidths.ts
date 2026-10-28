export interface EventColumnWidths {
    time: number;
    sensor: number;
    source: number;
    port: number;
    handler: number;
    scanner: number;
    end: number;
    frames: number;
    details: number;
}

export const DEFAULT_COLUMN_WIDTHS: EventColumnWidths = {
    time: 75,
    sensor: 85,
    source: 200,
    port: 80,
    handler: 80,
    scanner: 140,
    end: 180,
    frames: 65,
    details: 65,
};

export const MIN_COLUMN_WIDTHS: Record<keyof EventColumnWidths, number> = {
    time: 60,
    sensor: 50,
    source: 80,
    port: 50,
    handler: 50,
    scanner: 50,
    end: 60,
    frames: 45,
    details: 45,
};

export function clampColumnWidth(col: keyof EventColumnWidths, width: number): number {
    const min = MIN_COLUMN_WIDTHS[col] ?? 40;
    return Math.max(min, Math.round(width));
}

export function loadColumnWidths(storage?: Pick<Storage, 'getItem'>): EventColumnWidths {
    const store = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined);
    if (!store) {
        return { ...DEFAULT_COLUMN_WIDTHS };
    }
    try {
        const raw = store.getItem('eventColumnWidths');
        if (raw) {
            const parsed = JSON.parse(raw);
            if (parsed && typeof parsed === 'object') {
                return {
                    time: clampColumnWidth(
                        'time',
                        Number(parsed.time) || DEFAULT_COLUMN_WIDTHS.time,
                    ),
                    sensor: clampColumnWidth(
                        'sensor',
                        Number(parsed.sensor) || DEFAULT_COLUMN_WIDTHS.sensor,
                    ),
                    source: clampColumnWidth(
                        'source',
                        Number(parsed.source) || DEFAULT_COLUMN_WIDTHS.source,
                    ),
                    port: clampColumnWidth(
                        'port',
                        Number(parsed.port) || DEFAULT_COLUMN_WIDTHS.port,
                    ),
                    handler: clampColumnWidth(
                        'handler',
                        Number(parsed.handler) || DEFAULT_COLUMN_WIDTHS.handler,
                    ),
                    scanner: clampColumnWidth(
                        'scanner',
                        Number(parsed.scanner) || DEFAULT_COLUMN_WIDTHS.scanner,
                    ),
                    end: clampColumnWidth('end', Number(parsed.end) || DEFAULT_COLUMN_WIDTHS.end),
                    frames: clampColumnWidth(
                        'frames',
                        Number(parsed.frames) || DEFAULT_COLUMN_WIDTHS.frames,
                    ),
                    details: clampColumnWidth(
                        'details',
                        Number(parsed.details) || DEFAULT_COLUMN_WIDTHS.details,
                    ),
                };
            }
        }
    } catch {
        // ignore parse or storage errors
    }
    return { ...DEFAULT_COLUMN_WIDTHS };
}

export function saveColumnWidths(
    widths: EventColumnWidths,
    storage?: Pick<Storage, 'setItem'>,
): void {
    const store = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined);
    if (!store) return;
    try {
        store.setItem('eventColumnWidths', JSON.stringify(widths));
    } catch {
        // ignore storage errors
    }
}

export function buildEventColsStyle(widths: EventColumnWidths, hasCheck = false): string {
    const cols = [
        `${widths.time}px`,
        `${widths.sensor}px`,
        `${widths.source}px`,
        `${widths.port}px`,
        `${widths.handler}px`,
        `${widths.scanner}px`,
        `${widths.end}px`,
        `${widths.frames}px`,
        `${widths.details}px`,
    ];
    if (hasCheck) {
        cols.unshift('28px');
    }
    return cols.join(' ');
}

export function hasSavedColumnWidths(storage?: Pick<Storage, 'getItem'>): boolean {
    const store = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined);
    if (!store) return false;
    try {
        return !!store.getItem('eventColumnWidths');
    } catch {
        return false;
    }
}
