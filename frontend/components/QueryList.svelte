<script lang="ts">
    import { onMount } from 'svelte';
    import { token, userQueries } from '../store';
    import { Query, deleteQuery, getQueries, updateQuery } from '../query';
    import QueryModal from './QueryModal.svelte';

    let saveModal: QueryModal;

    onMount(() => {
        reloadQueries();
    });

    async function reloadQueries() {
        let updatedQueries = await getQueries($token);
        userQueries.set(updatedQueries);
    }

    async function deleteQueryAndReload(id: string) {
        await deleteQuery(id, $token);
        await reloadQueries();
    }

    async function toggleActive(query: Query) {
        console.log('toggling active in query', query);

        await updateQuery(
            {
                id: query.id,
                active: !query.active,
            },
            $token,
        );
        await reloadQueries();
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
                        {#if query.active}active{:else}inactive{/if}
                    </p>
                </div>
            </div>
            <div class="queryList__buttons-container">
                <QueryModal bind:this={saveModal} />
                <button on:click={() => saveModal.showModal(query)}>edit</button>
                <button on:click={() => deleteQueryAndReload(query.id)}>Delete</button>
                <button on:click={() => toggleActive(query)}>Toggle</button>
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
