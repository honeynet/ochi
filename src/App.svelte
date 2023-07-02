<script lang="ts">
    import Header from './components/Header.svelte';
    import MessageList from './components/MessageList.svelte';
    import Filter from './components/Filter.svelte';
    import Config from './components/Config.svelte';
    import Content from './components/Content.svelte';

    import { onDestroy, onMount } from 'svelte';
    import { ENV_DEV, ENV_PROD } from './constants';
    import type { Event } from './event';
    import { generateRandomTestEvent } from './util';
    import { validate } from './session';
    import { isAuthenticated, env } from './store';

    // subscribe to the authentication status
    let isLoggedIn: boolean;
    const isAuthenticatedUnsubscribe = isAuthenticated.subscribe((status) => {
        isLoggedIn = status;
    });

    let conn: WebSocket = null;
    let messageList: MessageList;

    function addMessage(message: Event) {
        messageList?.onNewMessage(message);
    }

    function dial(conn: WebSocket) {
        if ($env == ENV_DEV) {
            return;
        }
        let wsUrl =
            location.protocol === 'https:'
                ? `wss://${location.host}/subscribe`
                : `ws://${location.host}/subscribe`;
        conn = new WebSocket(wsUrl);

        if (conn) {
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
        }
        return true;
    }

    const sleep = (ms: number) => new Promise((f) => setTimeout(f, ms));

    const test = async () => {
        while ($env == ENV_DEV) {
            addMessage(generateRandomTestEvent());
            await sleep(1000);
        }
    };

    const envUnsubscribe = env.subscribe((value) => {
        if (value === ENV_DEV) {
            test();
        } else if (value === ENV_PROD) {
            dial(conn);
        }
    });

    onDestroy(() => {
        envUnsubscribe();
        isAuthenticatedUnsubscribe();
    });

    onMount(() => {
        validate();
    });
</script>

<Header {isLoggedIn} />
<main>
    <Filter />
    <Config />
    <div class="row">
        <MessageList bind:this={messageList} />
        <Content />
    </div>
</main>

<style>
    main {
        width: 100vw;
        min-width: 320px;
    }

    .row {
        margin-top: 100px;
        display: flex;
        position: absolute;
        top: 55px;
        left: 0;
        bottom: 0;
        right: 0;
    }
</style>
