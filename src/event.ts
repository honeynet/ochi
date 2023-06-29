export interface Event {
    payload?: string; // base64 encoded binary payload
    connKey?: number[]; // identifier based on IP and source port
    dstPort: number; // the connection destination port
    rule?: string; // the rule that matched the connection
    handler?: string; // the processing handler
    scanner?: string; // name of the scanner if detected
    sensorID: string; // the id of the sensor
    srcHost: string; // the source IP address
    srcPort: string; // the source port
    timestamp: string; // the UTC timestamp of the connection
    decoded?: any; // a decoded version of the payload if available
}
