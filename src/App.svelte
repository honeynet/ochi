<script lang="ts">
	import { onMount } from "svelte";
	import { now } from "svelte/internal";

	import Content from "./Content.svelte";
	import Message from "./Message.svelte";

	export let messages: messageType[] = [];

	$: if (messages.length >= 50) {
		messages = messages.slice(1);
	}

	let content: messageType;
	let filter: string;

	let filterPorts: number[] = [];

	function addMessage(message: messageType) {
		if (message.dstPort === null || !filterPorts.includes(message.dstPort)) {
			messages.push(message);
			messages = messages;
		}
	}

	function filterMessages() {
		filterPorts = filter.split("&&").map(Number);
		messages = messages.filter(message => !filterPorts.includes(message.dstPort));
	}

	function dial() {
		let conn: WebSocket;
		try {
			conn = new WebSocket(`wss://${location.host}/subscribe`);
		} catch (e) {
			conn = new WebSocket(`ws://${location.host}/subscribe`);
		}

		conn.addEventListener("close", (ev) => {
			if (ev.code !== 1001) {
				//appendLog("Reconnecting in 1s");
				setTimeout(dial, 1000);
			}
		});
		conn.addEventListener("open", (ev) => {
			console.info("websocket connected");
		});
		conn.addEventListener("message", (ev) => {
			const obj = JSON.parse(ev.data);
			console.log(obj);
			addMessage(obj);
		});
	}
	function displayContent(event) {
		content = event.detail;
	}

	const sleep = (ms: number) => new Promise((f) => setTimeout(f, ms));

	const test = async () => {
		while (true) {
			await sleep(1000);
			addMessage({
				action: "action",
				connKey: [2, 2],
				dstPort: 1234,
				rule: "TCP",
				scanner: "censys",
				sensorID: "sensorID",
				srcHost: "1.1.1.1",
				srcPort: "4321",
				timestamp: now().toString(),
				payload: "dGVzdA==", // test
			});
		}
	};

	onMount(() => {
		dial();
		//test();
	});
</script>

<header class="site-header">
	<b>Ochi</b>: find me at
	<a target="_blank" href="https://github.com/glaslos/ochi">github/glaslos/ochi</a>
	<input bind:value={filter} placeholder="Filter destination port" />
	<button on:click={filterMessages}>Apply</button>
	<span>Port number and '&&' to concat.</span>
</header>

<main>
	<div class="row">
		<div class="column" id="message-log">
			{#each messages as message (message.timestamp)}
				{#if !filterPorts.includes(message.dstPort)}
					<Message on:message={displayContent} {message} />
				{/if}
			{/each}
		</div>
		<Content {content} />
	</div>
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
</style>
