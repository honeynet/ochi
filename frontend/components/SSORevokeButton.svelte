<svelte:options runes={true} />

<script lang="ts">
    import { user } from '../store';
    import { logout } from '../session';
    import { showError } from '../toast';

    let revoking = $state(false);
    let email = $derived($user && typeof $user.email === 'string' ? $user.email : '');

    function revokeSSO() {
        if (!email) {
            showError('Cannot revoke SSO without an email');
            return;
        }

        if (typeof google === 'undefined' || !google.accounts?.id) {
            showError('Google services are still loading. Please try again.');
            return;
        }

        revoking = true;
        try {
            google.accounts.id.revoke(email, (response: google.accounts.id.RevocationResponse) => {
                revoking = false;
                if (response.successful) {
                    logout();
                } else {
                    showError(response.error ?? 'Failed to revoke SSO', 'Revocation error');
                }
            });
        } catch (e) {
            revoking = false;
            showError(e instanceof Error ? e.message : String(e), 'Failed to revoke SSO');
        }
    }
</script>

<button id="revokeButton" onclick={revokeSSO} disabled={revoking}>
    {revoking ? 'Revoking...' : 'Revoke'}
</button>

<style>
    #revokeButton {
        margin: 0;
    }
</style>
