/// <reference types="svelte" />

interface messageType {
    payload?: string;
    action?: string;
    connKey?: number[2];
    dstPort: number;
    rule?: string;
    scanner?: string;
    sensorID: string;
    srcHost: string;
    srcPort: string;
    timestamp: string;
}
