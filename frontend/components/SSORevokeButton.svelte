<script lang="ts">
    import { user } from '../store';
    import { logout } from '../session';

    let email = $derived($user && typeof $user.email === 'string' ? $user.email : '');

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
</script>

<button id="revokeButton" onclick={revokeSSO}>Revoke</button>

<style>
    #revokeButton {
        margin: 0;
    }
</style>
