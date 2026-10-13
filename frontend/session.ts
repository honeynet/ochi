import { isAuthenticated, user, token, userSensors, observedSensorIds } from './store';
import { showError } from './toast';

import { API_ENDPOINT } from './constants';

let jwt = '';
token.subscribe((value) => {
    jwt = value ?? '';
});

export async function validate() {
    if (jwt !== '') {
        const res = await fetch(`${API_ENDPOINT}/api/session`, {
            method: 'GET',
            headers: { Authorization: `Bearer ${jwt}` },
        });

        if (res.ok) {
            const json = await res.json();
            login(json);
        } else {
            showError('Failed to validate session');
            logout();
        }
    }
}

export function logout() {
    isAuthenticated.set(false);
    user.set({});
    token.set('');
    userSensors.set([]);
    observedSensorIds.set([]);
}

export function login(data: any) {
    user.set(data['user']);
    token.set(data['token']);
    isAuthenticated.set(true);
}
