<svelte:options runes={true} />

<script lang="ts">
    import PageShell from '../components/PageShell.svelte';
    import UserList from '../components/UserList.svelte';
</script>

<PageShell path="/" pathText="Go back">
    <UserList />
</PageShell>
