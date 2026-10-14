<script lang="ts">
    import { user } from '../store';
    import { logout } from '../session';
    import { showError } from '../toast';

    let email = $derived($user && typeof $user.email === 'string' ? $user.email : '');

    function revokeSSO() {
        if (!email) {
            showError('Cannot revoke SSO without an email');
            return;
        }
        if (typeof google === 'undefined' || !google.accounts?.id) {
            logout();
            return;
        }
        google.accounts.id.revoke(email, (response: google.accounts.id.RevocationResponse) => {
            if (response.successful) {
                logout();
            } else {
                showError(response.error, 'Failed to revoke SSO');
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
