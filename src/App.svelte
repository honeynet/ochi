<script lang="ts">
	import { onMount } from "svelte";

	import Content from "./Content.svelte";
	import Message from "./Message.svelte";

	export let messages: messageType[] = [];

	let content: messageType;

	$: if (messages.length >= 50) {
		messages = messages.slice(1);
	}

	function addMessage(obj: messageType) {
		messages.push(obj);
		messages = messages;
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

	function test() {
		addMessage({
			action: "action",
			connKey: [2, 2],
			dstPort: 1234,
			rule: "TCP",
			scanner: "censys",
			sensorID: "sensorID",
			srcHost: "1.1.1.1",
			srcPort: "4321",
			timestamp: "2022-05-23T21:02:26.882473842Z",
			payload: "dGVzdA==", // test
		});
	}

	onMount(() => {
		dial();
		//test();
	});
</script>

<main>
	<div class="row">
		<div class="column" id="message-log">
			{#each messages as message (message.timestamp)}
				<Message on:message={displayContent} {message} />
			{/each}
		</div>
		<Content {content} />
	</div>
</main>

<style>
	main {
		width: 100vw;
		min-width: 320px;
	}

	.row {
		display: flex;
		position: absolute;
		top: 0;
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
