<script lang="ts">
    import { logout } from "./session";
</script>

<button id="logoutButton" on:click={logout}>Logout</button>

<style>
    #logoutButton {
        float: right;
        margin-left: 0.5em;
    }
</style>
