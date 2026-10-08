<svelte:options runes={true} />

<script lang="ts">
    import PageShell from '../components/PageShell.svelte';
    import EventList from '../components/EventList.svelte';
</script>

<PageShell path="/" pathText="Go back">
    <EventList />
</PageShell>
