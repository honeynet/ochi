<svelte:options runes={true} />

<script lang="ts">
    import { Router } from '@roxi/routify';
    import { routes } from '../.routify/routes';
    import Toasts from './components/Toasts.svelte';
    import { showError } from './toast';

    const routifyConfig = {};
</script>

<svelte:window
    onunhandledrejection={(e) => showError(e.reason)}
    onerror={(e) => {
        const { error, message } = e as unknown as ErrorEvent;
        showError(error ?? message);
    }}
/>

<Router config={routifyConfig} {routes} />
<Toasts />
