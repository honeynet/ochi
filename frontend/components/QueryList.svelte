<script lang="ts">
    import { onMount } from 'svelte';
    import { token, userQueries, stringFilter, activeFilterId } from '../store';
    import { type Query, deleteQuery, getQueries } from '../query';
    import QueryModal from './QueryModal.svelte';
    import { showError } from '../toast';

    let saveModal = $state<{ showModal: (query: Query) => void } | null>(null);

    onMount(() => {
        reloadQueries();
    });

    async function reloadQueries() {
        const updatedQueries = await getQueries($token);
        userQueries.set(updatedQueries);
    }

    async function deleteQueryAndReload(id?: string) {
        if (!id) {
            showError('Cannot delete query without an id');
            return;
        }
        await deleteQuery(id, $token);
        await reloadQueries();
    }

    async function activate(query: Query) {
        stringFilter.set(query.content ?? '');
        activeFilterId.set(query.id ?? undefined);
    }

    async function deactivate(query: Query) {
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
                    {#if query.tags && query.tags.length > 0}
                        <ul class="queryList__tags" aria-label="Tags">
                            {#each query.tags as tag (JSON.stringify([tag.type, tag.value]))}
                                <li class="tag">{tag.type}: {tag.value}</li>
                            {/each}
                        </ul>
                    {/if}
                    <p class="queryList__info-field queryList__active">
                        {#if query.id == $activeFilterId}active{:else}inactive{/if}
                    </p>
                </div>
            </div>
            <div class="queryList__buttons-container">
                <button onclick={() => saveModal?.showModal(query)}>edit</button>
                <button onclick={() => deleteQueryAndReload(query.id)}>Delete</button>
                {#if $activeFilterId != query.id}
                    <button onclick={() => activate(query)}>Activate</button>
                {:else}
                    <button onclick={() => deactivate(query)}>Deactivate</button>
                {/if}
            </div>
        </li>
    {/each}
</ul>

<QueryModal bind:this={saveModal} />

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

    .queryList__tags {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .tag {
        padding: 0 8px;
        border-radius: 999px;
        background: #e8e8e8;
        font-size: 13px;
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
