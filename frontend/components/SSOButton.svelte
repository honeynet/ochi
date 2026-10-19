<svelte:options runes={true} />

<script module lang="ts">
    import { login } from '../session';
    import { API_ENDPOINT } from '../constants';
    import { showError } from '../toast';

    let isSSOInitialized = false;

    async function sendCredentials(data: google.accounts.id.CredentialResponse) {
        const res = await fetch(`${API_ENDPOINT}/api/login`, {
            method: 'POST',
            body: data.credential,
        });

        if (res.ok) {
            const json = await res.json();
            login(json);
        } else {
            showError('Failed to log in');
        }
    }

    function handleCredentialResponse(response: google.accounts.id.CredentialResponse) {
        if (response && response.credential) {
            sendCredentials(response);
        } else {
            showError('Invalid credential response');
        }
    }

    function ensureSSOInitialized(): boolean {
        if (isSSOInitialized) {
            return true;
        }
        if (typeof google === 'undefined' || !google.accounts?.id) {
            return false;
        }
        google.accounts.id.initialize({
            client_id: '610036027764-0lveoeejd62j594aqab5e24o2o82r8uf.apps.googleusercontent.com',
            ux_mode: 'popup',
            callback: handleCredentialResponse,
        });
        isSSOInitialized = true;
        return true;
    }
</script>

<script lang="ts">
    import { onMount } from 'svelte';

    let buttonElement = $state<HTMLElement | null>(null);

    function renderGoogleButton() {
        if (!buttonElement || typeof google === 'undefined' || !google.accounts?.id) {
            return;
        }
        if (!ensureSSOInitialized()) {
            return;
        }
        buttonElement.innerHTML = '';
        google.accounts.id.renderButton(buttonElement, {
            type: 'icon',
            size: 'small',
        });
    }

    function handleFallbackClick() {
        if (typeof google !== 'undefined' && google.accounts?.id) {
            ensureSSOInitialized();
            google.accounts.id.prompt();
        } else {
            showError('Google Sign-In is still loading. Please try again.');
        }
    }

    onMount(() => {
        if (typeof google !== 'undefined' && google.accounts?.id) {
            renderGoogleButton();
        } else {
            const existingScript = document.querySelector<HTMLScriptElement>(
                'script[src*="accounts.google.com/gsi/client"]',
            );
            if (existingScript) {
                existingScript.addEventListener('load', renderGoogleButton, { once: true });
            }
        }
    });

    $effect(() => {
        if (buttonElement && typeof google !== 'undefined' && google.accounts?.id) {
            renderGoogleButton();
        }
    });
</script>

<div bind:this={buttonElement} class="googleButton">
    <button
        class="googleButtonFallback"
        type="button"
        onclick={handleFallbackClick}
        title="Sign in with Google"
        aria-label="Sign in with Google"
    >
        <svg
            version="1.1"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 48 48"
            class="googleIcon"
        >
            <g>
                <path
                    fill="#EA4335"
                    d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
                ></path>
                <path
                    fill="#4285F4"
                    d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
                ></path>
                <path
                    fill="#FBBC05"
                    d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
                ></path>
                <path
                    fill="#34A853"
                    d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
                ></path>
                <path fill="none" d="M0 0h48v48H0z"></path>
            </g>
        </svg>
    </button>
</div>

<style>
    .googleButton {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        padding: 0;
        border: none;
        background: transparent;
        min-height: 32px;
    }

    .googleButtonFallback {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        padding: 4px;
        border: 1px solid #dadce0;
        border-radius: 4px;
        background: #fff;
        cursor: pointer;
        margin: 0;
        width: 32px;
        height: 32px;
        box-sizing: border-box;
    }

    .googleButtonFallback:hover {
        background: #f8f9fa;
        border-color: #d2e3fc;
    }

    .googleIcon {
        width: 18px;
        height: 18px;
    }
</style>
