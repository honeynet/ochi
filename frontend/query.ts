import { API_ENDPOINT } from './constants';
import { showError } from './toast';

export interface Query {
    id?: string; // query id
    content?: string; //query content
    owner_id?: string; // user id
    active?: boolean; // TODO: currently unused
    description?: string; // description of the query
}

export async function saveQuery(queryToEdit: Query, token: string) {
    const res = await fetch(`${API_ENDPOINT}/api/queries`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            content: queryToEdit.content,
            description: queryToEdit.description,
            active: queryToEdit.active,
        }),
    });

    if (!res.ok) {
        showError(`Failed to save query: ${await res.text()}`);
    }
}

export async function updateQuery(queryToEdit: Query, token: string): Promise<void> {
    const res = await fetch(`${API_ENDPOINT}/api/queries/${queryToEdit.id}`, {
        method: 'PATCH',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            id: queryToEdit.id,
            content: queryToEdit.content,
            description: queryToEdit.description,
            active: queryToEdit.active,
        }),
    });

    if (!res.ok) {
        throw new Error('Could not update a query');
    }
}

export async function getQueries(token: string): Promise<Query[]> {
    const res = await fetch(`${API_ENDPOINT}/api/queries`, {
        method: 'GET',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
    });

    if (res.ok) {
        return await res.json();
    } else {
        throw new Error('Could not fetch queries');
    }
}

export async function deleteQuery(id: string, token: string): Promise<void> {
    const res = await fetch(`${API_ENDPOINT}/api/queries/${id}`, {
        method: 'DELETE',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
    });

    if (!res.ok) {
        throw new Error('Could not delete a query');
    }
}

export async function createQuery(queryToEdit: Query, token: string): Promise<Query> {
    const res = await fetch(`${API_ENDPOINT}/api/queries`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            content: queryToEdit.content,
            description: queryToEdit.description,
            active: queryToEdit.active,
        }),
    });

    if (res.ok) {
        return await res.json();
    } else {
        throw new Error('Could not create a query');
    }
}
