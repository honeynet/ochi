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
}

export function formatDest(event: Event): string {
    if (event.dstHost) {
        return `${event.dstHost}:${event.dstPort}`;
    }
    return `:${event.dstPort}`;
}

export function displayRule(event: Event): string | undefined {
    return event.ruleName || event.rule;
}
