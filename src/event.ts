export interface Event {
    payload?: string;
    action?: string;
    connKey?: number[];
    dstPort: number;
    rule?: string;
    scanner?: string;
    sensorID: string;
    srcHost: string;
    srcPort: string;
    timestamp: string;
}
