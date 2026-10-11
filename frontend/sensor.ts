import { API_ENDPOINT } from './constants';

export interface Sensor {
    id: string; // sensor id
    name: string; // user-given sensor name
    user_id: string; // owner id
}

export async function getSensors(token: string): Promise<Sensor[]> {
    const res = await fetch(`${API_ENDPOINT}/api/sensors`, {
        headers: { Authorization: `Bearer ${token}` },
    });
    if (res.status === 404) {
        return [];
    }
    if (!res.ok) {
        throw new Error(`failed to load sensors: ${await res.text()}`);
    }
    return (await res.json()) ?? [];
}

export async function addSensor(token: string, sensor: Pick<Sensor, 'id' | 'name'>): Promise<Sensor> {
    const res = await fetch(`${API_ENDPOINT}/api/sensors`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(sensor),
    });
    if (!res.ok) {
        throw new Error(`failed to add sensor: ${await res.text()}`);
    }
    return await res.json();
}

export const DOWNLOAD_OS = 'linux';
export const DOWNLOAD_ARCH = 'amd64';

// Unauthenticated, direct download URL (usable with curl). The sensor id in the path and the owner's
// user id in the query string are both secrets, so the link must not be shared.
export function sensorDownloadUrl(sensor: Sensor): string {
    const path = `${API_ENDPOINT}/api/download/${encodeURIComponent(sensor.id)}`;
    const params = new URLSearchParams({ os: DOWNLOAD_OS, arch: DOWNLOAD_ARCH, user_id: sensor.user_id });
    return new URL(`${path}?${params}`, window.location.href).toString();
}
