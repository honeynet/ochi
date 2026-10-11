<svelte:options runes={true} />

<script lang="ts">
    import PageShell from '../components/PageShell.svelte';
    import SensorList from '../components/SensorList.svelte';
</script>

<PageShell path="/" pathText="Go back">
    <SensorList />
</PageShell>
