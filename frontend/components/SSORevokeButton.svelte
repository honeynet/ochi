<script lang="ts">
    import { onDestroy } from 'svelte';
    import { user } from '../store';
    import { logout } from '../session';
    import 'google.accounts';

    let email = '';
    const unsubscribe = user.subscribe((value) => {
        email = value && typeof value.email === 'string' ? value.email : '';
    });

    function revokeSSO() {
        if (!email) {
            console.warn('Cannot revoke SSO without an email');
            return;
        }
        google.accounts.id.revoke(email, (response: google.accounts.id.RevocationResponse) => {
            if (response.successful) {
                logout();
            } else {
                console.log(response.error);
            }
        });
    }

    onDestroy(() => {
        unsubscribe();
    });
</script>

<button id="revokeButton" on:click={revokeSSO}>Revoke</button>

<style>
    #revokeButton {
        float: right;
        margin: 0;
    }
</style>
