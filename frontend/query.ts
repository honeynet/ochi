import { API_ENDPOINT } from './constants';

export interface Query {
    id?: string; // query id
    content?: string; //query content
    owner_id?: string; // user id
    active?: boolean; // TODO: currently unused
    description?: string; // description of the query
}

export async function saveQuery(queryToEdit: Query, token: string) {
    console.log('saving query');
    const res = await fetch(`${API_ENDPOINT}/queries`, {
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
        console.log('received success ' + res.text());
    } else {
        console.log('failed to save ' + res.text());
    }
}

export async function updateQuery(queryToEdit: Query, token: string): Promise<void> {
    console.log('updating a query');
    const res = await fetch(`${API_ENDPOINT}/queries/${queryToEdit.id}`, {
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

    if (res.ok) {
        console.log('received success');
    } else {
        console.log('failed to update');
        throw new Error('Could not update a query');
    }
}

export async function getQueries(token: string): Promise<Query[]> {
    const res = await fetch(`${API_ENDPOINT}/queries`, {
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
    console.log(`deleting query with id ${id}`);
    const res = await fetch(`${API_ENDPOINT}/queries/${id}`, {
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
    console.log('creating a query', queryToEdit);
    const res = await fetch(`${API_ENDPOINT}/queries`, {
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
