<script lang="ts">
    import { isActive } from '@roxi/routify';
    import Home from '../components/Home.svelte';

    // Routify passes page content through a legacy slot (with let: directives),
    // so this module can't render it as a `children` snippet.
</script>

<!-- Home stays mounted so the websocket and message list survive navigation. -->
<Home active={$isActive('/index')} />
<slot />
