<svelte:options runes={true} />

<script lang="ts">
    import { logout } from '../session';
</script>

<button id="logoutButton" onclick={logout}>Logout</button>

<style>
    #logoutButton {
        margin: 0;
    }
</style>
