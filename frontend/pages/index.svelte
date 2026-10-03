<script lang="ts">
    import { metatags } from '@roxi/routify';

    metatags.title = 'Ochi';
    metatags.description = 'Web UI for honeypot events';
    import PageShell from '../components/PageShell.svelte';
    import ColumnSplit from '../components/ColumnSplit.svelte';
    import MessageList from '../components/MessageList.svelte';
    import Filter from '../components/Filter.svelte';
    import Content from '../components/Content.svelte';

    import { onMount, untrack } from 'svelte';
    import { ENV_DEV, ENV_PROD, WS_ENDPOINT } from '../constants';
    import type { Event } from '../event';
    import { generateRandomTestEvent } from '../util';
    import { validate } from '../session';
    import { env } from '../store';

    let conn: WebSocket | null = null;
    let messageList: MessageList | null = $state(null);

    function addMessage(message: Event) {
        messageList?.onNewMessage(message);
    }

    function dial() {
        if ($env == ENV_DEV) {
            return;
        }
        let wsUrl = `${WS_ENDPOINT}/subscribe`;
        // location.protocol === 'https:'
        //     ? `wss://${location.host}/subscribe`
        //     : `ws://${location.host}/subscribe`;
        conn = new WebSocket(wsUrl);

        conn.addEventListener('close', (ev) => {
            if (ev.code !== 1001) {
                setTimeout(dial, 1000);
            }
        });
        conn.addEventListener('open', () => {
            console.info('websocket connected');
        });
        conn.addEventListener('message', (ev) => {
            const obj = JSON.parse(ev.data);
            console.log(obj);
            addMessage(obj);
        });
        return true;
    }

    const sleep = (ms: number) => new Promise((f) => setTimeout(f, ms));

    const test = async () => {
        while ($env == ENV_DEV) {
            addMessage(generateRandomTestEvent());
            await sleep(1000);
        }
    };

    $effect(() => {
        const mode = $env;
        untrack(() => {
            if (mode === ENV_DEV) {
                test();
            } else if (mode === ENV_PROD) {
                dial();
            }
        });
    });

    onMount(() => {
        validate();
    });
</script>

<PageShell path="/myqueries" pathText="My Queries">
    {#snippet headerCenter()}
        <Filter />
    {/snippet}
    <main>
        <ColumnSplit>
            {#snippet left()}
                <MessageList bind:this={messageList} />
            {/snippet}
            {#snippet right()}
                <Content isShared={false} />
            {/snippet}
        </ColumnSplit>
    </main>
</PageShell>

<style>
    main {
        display: flex;
        flex-direction: column;
        flex: 1;
        min-width: 320px;
        min-height: 0;
        width: 100%;
    }
</style>
