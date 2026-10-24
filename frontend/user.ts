import { API_ENDPOINT } from './constants';

// Must match the permissions in backend/entities/role.go.
export const PERM_LIST_USERS = 'users:list';
export const PERM_ADD_SENSOR_BY_ID = 'sensors:add_by_id';

export interface User {
    id: string;
    email: string;
    roles?: string[];
}

export async function listUsers(token: string): Promise<User[]> {
    const res = await fetch(`${API_ENDPOINT}/api/users`, {
        headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
        throw new Error(`failed to load users: ${await res.text()}`);
    }
    return (await res.json()) ?? [];
}
