<svelte:options runes={true} />

<script lang="ts">
    import PageShell from '../components/PageShell.svelte';
    import QueryList from '../components/QueryList.svelte';
</script>

<PageShell path="/" pathText="Go back">
    <QueryList />
</PageShell>
