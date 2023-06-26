export interface Event {
    payload?: string;
    connKey?: number[];
    dstPort: number;
    rule?: string;
    handler?: string;
    scanner?: string;
    sensorID: string;
    srcHost: string;
    srcPort: string;
    timestamp: string;
    decoded?: any;
}
