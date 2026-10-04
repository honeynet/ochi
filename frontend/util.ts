import type { Event } from './event';

/**
 * Debounces a callback to prevent calling it too many times.
 *
 * The wrapped function is called only after `delay` milliseconds after
 * the last call to this function. This wrapper is useful for debouncing
 * in UI widgets like text inputs.
 * @param callback Function to be called
 * @param delay Milliseconds to wait after last event before the function is called
 * @returns
 */
export function debounce<T extends (...args: any[]) => void>(callback: T, delay: number): T {
    let timeoutId: NodeJS.Timeout | undefined = undefined;

    return <T>((...args: any[]): void => {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => {
            callback(...args);
        }, delay);
    });
}

const handlers: (string | undefined)[] = ['http', 'rdp', '', undefined];

function generateRandomString(length: number): string {
    let result = '';
    const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    const charactersLength = characters.length;
    let counter = 0;
    while (counter < length) {
        result += characters.charAt(Math.floor(Math.random() * charactersLength));
        counter += 1;
    }
    return result;
}

function generateUUID() {
    // Public Domain/MIT
    var d = new Date().getTime(); //Timestamp
    var d2 =
        (typeof performance !== 'undefined' && performance.now && performance.now() * 1000) || 0; //Time in microseconds since page-load or 0 if unsupported
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = Math.random() * 16; //random number between 0 and 16
        if (d > 0) {
            //Use timestamp until depleted
            r = (d + r) % 16 | 0;
            d = Math.floor(d / 16);
        } else {
            //Use microseconds since page-load if supported
            r = (d2 + r) % 16 | 0;
            d2 = Math.floor(d2 / 16);
        }
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
    });
}

/**
 * Generates a random event used for UI testing
 * @returns test event
 */
export function generateRandomTestEvent(): Event {
    const payloadText = `test ${generateRandomString(10 + Math.floor(Math.random() * 40))}`;
    const payload = btoa(payloadText);
    const kinds = ['smb', 'http', 'rdp', 'tcp'] as const;
    const kind = kinds[Math.floor(Math.random() * kinds.length)];
    const base: Event = {
        connKey: [2, 2],
        dstPort: 80,
        dstHost: '198.51.100.8',
        rule: 'Rule: TCP',
        scanner: Math.random() > 0.5 ? 'censys' : undefined,
        sensorID: generateUUID().split('-')[0],
        sensorVersion: Math.random() > 0.3 ? '1.2.3' : undefined,
        srcHost: '203.0.113.10',
        srcPort: '4321',
        srcPtr: 'scanner.example.',
        timestamp: new Date().toISOString(),
        startedAt: new Date(Date.now() - 1500).toISOString(),
        durationMs: 1500,
        payload,
        payloadHash: 'deadbeef',
        frameCount: 1,
        endReason: 'timeout',
        handler: kind,
    };
    if (kind === 'smb') {
        return {
            ...base,
            dstPort: 445,
            rule: 'Rule: SMB',
            ruleName: 'smb',
            frameCount: 2,
            decoded: [
                {
                    direction: 'read',
                    path: 'IPC$',
                    setup: 'TRANS2_SESSION_SETUP',
                    status: 'STATUS_NOT_IMPLEMENTED',
                    header: { tid: 0, uid: 1, mid: 2, pid: 3, flags2: '0xc053' },
                    payload,
                },
                {
                    direction: 'write',
                    path: 'IPC$',
                    status: 'STATUS_SUCCESS',
                    header: { flags2: [83, 192] },
                    payload,
                },
            ],
        };
    }
    if (kind === 'http') {
        return {
            ...base,
            dstPort: 80,
            rule: 'Rule: HTTP',
            ruleName: 'http',
            decoded: [
                {
                    direction: 'read',
                    command: 'GET',
                    path: '/',
                    host: 'example.test',
                    user_agent: 'curl/8.0',
                    payload,
                },
                {
                    direction: 'write',
                    status: 200,
                    session_id: 'abc',
                    payload,
                },
            ],
            frameCount: 2,
        };
    }
    if (kind === 'rdp') {
        return {
            ...base,
            dstPort: 3389,
            rule: 'Rule: RDP',
            ruleName: 'rdp',
            decoded: [
                {
                    direction: 'read',
                    command: 'ConnectionRequest',
                    cookie: 'hello',
                    protocols: ['tls'],
                    payload,
                },
            ],
        };
    }
    return {
        ...base,
        decoded: [
            {
                direction: 'read',
                payload_hash: 'deadbeef',
                truncated: false,
                payload,
            },
        ],
    };
}

/**
 * Generates an event used for testing
 * @returns test event
 */
export function generateTestEvent(
    dport: number,
    sport?: string,
    sip?: string,
    payload?: string,
    rule: string = 'Rule: TCP',
    extra: Partial<Event> = {},
): Event {
    return {
        handler: handlers[Math.floor(Math.random() * handlers.length)],
        connKey: [2, 2],
        dstPort: dport,
        rule: rule,
        scanner: 'censys',
        sensorID: 'sensorID',
        srcHost: sip ?? '',
        srcPort: sport ?? '1234',
        timestamp: new Date().toISOString(),
        payload: payload,
        decoded: { payload: 'test' },
        ...extra,
    };
}
