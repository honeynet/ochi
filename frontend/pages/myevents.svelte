<svelte:options runes={true} />

<script lang="ts">
    import { onMount } from 'svelte';
    import PageShell from '../components/PageShell.svelte';
    import ColumnSplit from '../components/ColumnSplit.svelte';
    import EventList from '../components/EventList.svelte';
    import Content from '../components/Content.svelte';
    import { currentEvent } from '../store';

    onMount(() => {
        currentEvent.set(undefined);
    });
</script>

<PageShell path="/" pathText="Go back">
    <main>
        <ColumnSplit>
            {#snippet left()}
                <EventList />
            {/snippet}
            {#snippet right()}
                <Content isShared={true} />
            {/snippet}
        </ColumnSplit>
    </main>
</PageShell>

<style>
    main {
        display: flex;
        flex-direction: column;
        flex: 1;
        min-width: 320px;
        min-height: 0;
        width: 100%;
    }
</style>
