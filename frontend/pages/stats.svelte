<svelte:options runes={true} />

<script lang="ts">
    import { onMount } from 'svelte';
    import PageShell from '../components/PageShell.svelte';
    import ConfirmDialog from '../components/ConfirmDialog.svelte';
    import { topEntries } from '../stats';
    import { stats, eventsLastMinute, resetStats } from '../statsStore';
    import { userQueries } from '../store';

    const TOP_N = 10;

    let now = $state(Date.now());
    let confirmDialog = $state<ConfirmDialog | null>(null);

    onMount(() => {
        // The rate is over a sliding minute, so it has to drop even when no events arrive.
        const timer = setInterval(() => (now = Date.now()), 1000);
        return () => clearInterval(timer);
    });

    let rate = $derived.by(() => {
        $stats;
        return eventsLastMinute(now);
    });
    let ports = $derived(topEntries($stats.ports, TOP_N));
    let handlers = $derived(topEntries($stats.handlers, TOP_N));
    // Only queries that still exist; a deleted query's count stays stored until a reset.
    let queries = $derived(
        $userQueries
            .filter((query) => query.id)
            .map((query) => ({
                label: query.description || query.content || '',
                count: $stats.queries.get(query.id!) ?? 0,
            }))
            .sort((a, b) => b.count - a.count),
    );

    function confirmReset() {
        resetStats();
        confirmDialog?.close();
    }
</script>

<PageShell path="/" pathText="Go back">
    <div class="stats">
        <h2>Statistics</h2>
        <p class="stats__summary">
            <span><strong id="stats-total">{$stats.total}</strong> events</span>
            <span><strong id="stats-rate">{rate}</strong> in the last minute</span>
            <span>since {new Date($stats.since).toLocaleString()}</span>
            <button class="stats__reset" onclick={() => confirmDialog?.showModal()}>Reset</button>
        </p>
        <p class="stats__note">Counted in this browser from the live events, while Ochi is open.</p>

        <div class="stats__tables">
            <section>
                <h3>Destination ports</h3>
                {#if ports.length === 0}
                    <p class="stats__empty">No events yet.</p>
                {:else}
                    <table id="stats-ports">
                        <tbody>
                            {#each ports as [key, count] (key)}
                                <tr><td>{key}</td><td>{count}</td></tr>
                            {/each}
                        </tbody>
                    </table>
                {/if}
            </section>

            <section>
                <h3>Handlers</h3>
                {#if handlers.length === 0}
                    <p class="stats__empty">No events yet.</p>
                {:else}
                    <table id="stats-handlers">
                        <tbody>
                            {#each handlers as [key, count] (key)}
                                <tr><td>{key}</td><td>{count}</td></tr>
                            {/each}
                        </tbody>
                    </table>
                {/if}
            </section>

            <section>
                <h3>Saved queries</h3>
                {#if queries.length === 0}
                    <p class="stats__empty">Sign in and save a query to count its matches.</p>
                {:else}
                    <table id="stats-queries">
                        <tbody>
                            {#each queries as query}
                                <tr><td>{query.label}</td><td>{query.count}</td></tr>
                            {/each}
                        </tbody>
                    </table>
                {/if}
            </section>
        </div>
    </div>
</PageShell>

<ConfirmDialog bind:this={confirmDialog} confirmText="Reset" onconfirm={confirmReset}>
    <p>Reset all statistics? The counts start again from zero.</p>
</ConfirmDialog>

<style>
    .stats {
        padding: 0 20px 20px;
    }

    .stats__summary {
        display: flex;
        flex-wrap: wrap;
        gap: 20px;
        align-items: center;
    }

    .stats__reset {
        margin: 0;
    }

    .stats__note,
    .stats__empty {
        color: #666;
    }

    .stats__tables {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
        gap: 20px;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    td {
        padding: 4px 8px;
        border-bottom: 1px solid #ddd;
    }

    td:last-child {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }
</style>
