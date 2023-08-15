<script lang="ts">
    import { onMount } from 'svelte';
    import { token } from '../store';
    import type { Query } from '../query';
    import { API_ENDPOINT } from '../constants';

    let queryList = getQueries();
    let queryToEdit = undefined;

    async function getQueries(): Promise<Query[]> {
        console.log('fetching queries');
        const res = await fetch(`${API_ENDPOINT}/queries`, {
            method: 'GET',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
        });

        if (res.ok) {
            console.log('received success ');
            const data = await res.json();
            console.log(data);
            return data;
        } else {
            console.log('failed to save ' + res.text());
            throw new Error('Could not fetch queries');
        }
    }

    async function updateQuery() {
        console.log('updating a query');
        const res = await fetch(`${API_ENDPOINT}/queries/${queryToEdit.id}`, {
            method: 'PATCH',
            headers: {
                Authorization: `Bearer ${$token}`,
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
            queryList = getQueries();
            queryToEdit = undefined;
        } else {
            console.log('failed to update');
            throw new Error('Could not update a query');
        }
    }

    function editQuery(query: Query) {
        queryToEdit = query;
    }

    function hideForm() {
        queryToEdit = undefined;
    }

    async function deleteQuery(id) {
        console.log(`deleting query with id ${id}`);
        const res = await fetch(`${API_ENDPOINT}/queries/${id}`, {
            method: 'DELETE',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
        });

        if (res.ok) {
            console.log('received success');
            queryList = getQueries();
        } else {
            console.log('failed to delete');
            throw new Error('Could not delete a query');
        }
    }
</script>

{#await queryList then data}
    <h2 class="queryList__title">Saved Queries</h2>
    <ul class="queryList__items">
        {#each data as query, index (query.id)}
            <li class="queryList__item">
                <div class="queryList__item-container">
                    {index + 1}.
                    <div class="queryList__info-container">
                        <p class="queryList__info-field queryList__content">{query.content}</p>
                        <p class="queryList__info-field queryList__description">
                            {#if query.description}{query.description}{/if}
                        </p>
                        <p class="queryList__info-field queryList__active">
                            {#if query.active}active{:else}inactive{/if}
                        </p>
                    </div>
                </div>
                <div class="queryList__buttons-container">
                    <button on:click={() => editQuery(query)}>Edit</button>
                    <button on:click={() => deleteQuery(query.id)}>Delete</button>
                </div>
            </li>
        {/each}
    </ul>
    {#if queryToEdit}
        <form class="queryList__form">
            <label>
                content
                <input name="content" type="text" bind:value={queryToEdit.content} required />
            </label>
            <label>
                description
                <input name="description" type="text" bind:value={queryToEdit.description} />
            </label>
            <label>
                active
                <input name="active" type="checkbox" bind:checked={queryToEdit.active} />
            </label>
            <button type="button" on:click={updateQuery}>Save</button>
            <button type="button" on:click={hideForm}>Cancel</button>
        </form>
    {/if}
{/await}

<style>
    .queryList__items {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 15px;
    }

    .queryList__item {
        width: 40%;
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
    }

    .queryList__item-container {
        display: flex;
        justify-content: flex-start;
        gap: 10px;
        font-size: 20px;
    }

    .queryList__info-field {
        margin: 0;
    }

    .queryList__content {
        font-size: 20px;
    }

    .queryList__description {
        font-style: italic;
        font-size: 14px;
    }

    .queryList__active {
        font-style: italic;
        font-size: 14px;
    }

    .queryList__info-container {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .queryList__buttons-container {
        align-self: center;
    }

    .queryList__title {
        text-align: center;
    }
</style>
