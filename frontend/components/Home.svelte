<script lang="ts">
    import PageShell from './PageShell.svelte';
    import ColumnSplit from './ColumnSplit.svelte';
    import MessageList from './MessageList.svelte';
    import Filter from './Filter.svelte';
    import Content from './Content.svelte';

    import { onMount, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { ENV_DEV, ENV_PROD, WS_ENDPOINT } from '../constants';
    import type { Event } from '../event';
    import { generateRandomTestEvent } from '../util';
    import { validate } from '../session';
    import { currentEvent, env } from '../store';
    import type { Event as OchiEvent } from '../event';

    let { active }: { active: boolean } = $props();

    let conn: WebSocket | null = null;
    let messageList: MessageList | null = $state(null);

    function addMessage(message: Event) {
        messageList?.onNewMessage(message);
    }

    function dial() {
        if ($env == ENV_DEV) {
            return;
        }
        let wsUrl = `${WS_ENDPOINT}/api/subscribe`;
        // location.protocol === 'https:'
        //     ? `wss://${location.host}/api/subscribe`
        //     : `ws://${location.host}/api/subscribe`;
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

    // Other pages overwrite currentEvent, so stash the selection while hidden.
    let stashed: OchiEvent | undefined;
    let wasActive = true;
    $effect(() => {
        const isActive = active;
        untrack(() => {
            if (isActive === wasActive) {
                return;
            }
            if (isActive) {
                currentEvent.set(stashed);
            } else {
                stashed = get(currentEvent);
            }
            wasActive = isActive;
        });
    });

    onMount(() => {
        currentEvent.set(undefined);
        validate();
    });
</script>

<div class="home" hidden={!active}>
    <PageShell path="/myqueries" pathText="Queries" showEvents>
        {#snippet headerCenter()}
            <Filter />
        {/snippet}
        <main>
            <ColumnSplit>
                {#snippet left()}
                    <MessageList bind:this={messageList} {active} />
                {/snippet}
                {#snippet right()}
                    <Content isShared={false} />
                {/snippet}
            </ColumnSplit>
        </main>
    </PageShell>
</div>

<style>
    .home {
        height: 100%;
    }

    .home[hidden] {
        display: none;
    }

    main {
        display: flex;
        flex-direction: column;
        flex: 1;
        min-width: 320px;
        min-height: 0;
        width: 100%;
    }
</style>
