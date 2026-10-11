<svelte:options runes={true} />

<script lang="ts">
    import { Router, urlRoute } from '@roxi/routify';
    import { routes } from '../.routify/routes';
    import Home from './components/Home.svelte';
    import Toasts from './components/Toasts.svelte';
    import { showError } from './toast';

    const routifyConfig = {};
    // Home stays mounted so the websocket and message list survive navigation.
    let homeActive = $derived(
        ['/', '/index'].includes(($urlRoute as { path?: string } | null)?.path ?? ''),
    );
</script>

<svelte:window
    onunhandledrejection={(e) => showError(e.reason)}
    onerror={(e) => {
        const { error, message } = e as unknown as ErrorEvent;
        showError(error ?? message);
    }}
/>

<Home active={homeActive} />
<Router config={routifyConfig} {routes} />
<Toasts />
