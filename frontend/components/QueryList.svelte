<script lang="ts">
    import { onMount } from 'svelte';
    import { token, userQueries, stringFilter, activeFilterId } from '../store';
    import { type Query, deleteQuery, getQueries } from '../query';
    import { parseDSL } from '../dsl';
    import QueryModal from './QueryModal.svelte';

    let saveModal: QueryModal;

    onMount(() => {
        reloadQueries();
    });

    async function reloadQueries() {
        const updatedQueries = await getQueries($token);
        userQueries.set(updatedQueries);
    }

    async function deleteQueryAndReload(id?: string) {
        if (!id) {
            console.warn('Cannot delete query without an id');
            return;
        }
        await deleteQuery(id, $token);
        await reloadQueries();
    }

    async function activate(query: Query) {
        console.log('activating query', query);

        stringFilter.set(query.content ?? '');
        activeFilterId.set(query.id ?? undefined);
    }

    async function deactivate(query: Query) {
        console.log('deactivating query', query);

        activeFilterId.set('');
        stringFilter.set('');
    }
</script>

<h2 class="queryList__title">Saved Queries</h2>
<ul class="queryList__items">
    {#each $userQueries as query, index (query.id)}
        <li class="queryList__item">
            <div class="queryList__item-container">
                {index + 1}.
                <div class="queryList__info-container">
                    <p class="queryList__info-field queryList__content">{query.content}</p>
                    <p class="queryList__info-field queryList__description">
                        {#if query.description}{query.description}{/if}
                    </p>
                    <p class="queryList__info-field queryList__active">
                        {#if query.id == $activeFilterId}active{:else}inactive{/if}
                    </p>
                </div>
            </div>
            <div class="queryList__buttons-container">
                <QueryModal bind:this={saveModal} />
                <button on:click={() => saveModal.showModal(query)}>edit</button>
                <button on:click={() => deleteQueryAndReload(query.id)}>Delete</button>
                {#if $activeFilterId != query.id}
                    <button on:click={() => activate(query)}>Activate</button>
                {:else}
                    <button on:click={() => deactivate(query)}>Deactivate</button>
                {/if}
            </div>
        </li>
    {/each}
</ul>

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
