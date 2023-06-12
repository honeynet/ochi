<script lang="ts">
    import { onMount } from 'svelte';

    import Modal from './Modal.svelte';
    import { ENV_DEV, ENV_PROD } from './Constants.svelte';

    import Content from './Content.svelte';
    import type { Event } from './event';
    import { debounce, generateRandomTestEvent } from './util';
    import { filterEvent } from './eventFilter';
    import type { QueryCstNode } from './generated/chevrotain_dts';
    import Message from './Message.svelte';
    import SSOButton from './SSOButton.svelte';
    import LogoutButton from './LogoutButton.svelte';
    import SSORevokeButton from './SSORevokeButton.svelte';
    import { validate } from './session';
    import { parseDSL } from './dsl';

    import { isAuthenticated } from './store';
    // subscribe to the authentication status
    let isLoggedIn: boolean;
    isAuthenticated.subscribe((status) => {
        isLoggedIn = status;
    });

    export let messages: Event[] = [];

    let content: Event;
    let configModal: Modal;
    let filter: string;
    let filterValid: boolean = false;
    let parsedFilter: QueryCstNode | undefined = undefined;
    let conn: WebSocket;
    let follow: boolean = true;
    let env: string;
    let maxNumberOfMessages: number;

    $: if (env == ENV_DEV) {
        test();
    } else if (env == ENV_PROD) {
        dial(conn);
    }

    let filterPorts: number[] = [];

    function addMessage(message: Event) {
        if (!parsedFilter || filterEvent(message, parsedFilter)) {
            messages.push(message);
            messages = messages;

            if (maxNumberOfMessages < messages.length) {
                messages = messages.slice(messages.length - maxNumberOfMessages);
            }
        }
    }

    function filterChangeHandler(): () => void {
        return debounce(() => {
            // TODO: validate queries as user types them.
            console.log('filter is changing');
            let parseResult = parseDSL(filter);
            if (parseResult.lexErrors.length > 0 || parseResult.parseErrors.length > 0) {
                console.log('Found some errors', parseResult.lexErrors, parseResult.parseErrors);
                filterValid = false;
                // TODO: highlight in red
            } else {
                filterValid = true;
            }
        }, 1000);
    }

    function applyFilter() {
        console.log(`Going to parse ${filter}`);
        let parseResult = parseDSL(filter);
        if (parseResult.lexErrors.length > 0) {
            console.error(parseResult.lexErrors);
            return;
        }
        if (parseResult.parseErrors.length > 0) {
            console.error(parseResult.parseErrors);
            return;
        }

        parsedFilter = parseResult.cst;
        messages = messages.filter((message) => filterEvent(message, parsedFilter));
    }

    function dial(conn: WebSocket) {
        if (env == ENV_DEV) {
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

    function displayContent(event: any) {
        content = event.detail;
    }

    const sleep = (ms: number) => new Promise((f) => setTimeout(f, ms));

    const test = async () => {
        while (env == ENV_DEV) {
            await sleep(1000);
            addMessage(generateRandomTestEvent());
        }
    };

    onMount(() => {
        // Default value of number of messages
        maxNumberOfMessages = 50;
        env = ENV_PROD;
        conn = null;
        validate();
    });

    function updateConfig() {
        if (maxNumberOfMessages <= 0) {
            return;
        }

        if (maxNumberOfMessages < messages.length) {
            messages = messages.slice(messages.length - maxNumberOfMessages, messages.length);
        }

        if (env == ENV_PROD) {
            if (conn != null) {
                conn.close();
            }
            messages = [];
        }
    }
</script>

<Modal bind:this={configModal} bind:env bind:maxNumberOfMessages on:configChange={updateConfig} />

<header class="site-header">
    <b>Ochi</b>: find me at
    <a target="_blank" href="https://github.com/glaslos/ochi">github/glaslos/ochi</a>
    <input
        class:input-error={filter && !filterValid}
        bind:value={filter}
        placeholder="Filter destination port"
        on:input={filterChangeHandler()}
    />
    <button disabled={!filterValid} on:click={applyFilter}>Apply</button>
    <span>Port number and '&&' to concat.</span>
    <button
        id="configButton"
        on:click={() => {
            configModal.showModal();
        }}>Config</button
    >
    {#if !isLoggedIn}
        <SSOButton />
    {:else}
        <LogoutButton />
        <SSORevokeButton />
    {/if}
</header>

<main>
    <div class="row">
        <div
            class="column"
            id="message-log"
            on:wheel={() => {
                follow = false;
            }}
        >
            {#each messages as message (message.timestamp)}
                {#if !filterPorts.includes(message.dstPort)}
                    <Message on:message={displayContent} {message} {follow} />
                {/if}
            {/each}
        </div>
        <Content {content} />
    </div>

    {#if !follow}
        <button
            on:click={() => {
                follow = true;
            }}
            id="resume-btn">Resume</button
        >
    {/if}
</main>

<style>
    .site-header {
        border-bottom-style: solid;
        border-width: 1px;
    }

    main {
        width: 100vw;
        min-width: 320px;
    }

    .row {
        display: flex;
        position: absolute;
        top: 55px;
        left: 0;
        bottom: 0;
        right: 0;
    }

    .column {
        flex: 50%;
        padding: 15px 20px;
    }

    #message-log {
        width: 100%;
        flex-grow: 1;
        overflow-y: scroll;
    }

    #resume-btn {
        position: fixed;
        bottom: 2rem;
        z-index: 2;
        left: 40vw;
        cursor: pointer;
    }

    .site-header input.input-error {
        border: 1px solid #ff0000;
    }
</style>
