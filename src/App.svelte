<script lang="ts">
	import { onMount } from "svelte";
	import { now } from "svelte/internal";

	import Content from "./Content.svelte";
	import Message from "./Message.svelte";
	import SSOButton from "./SOOButton.svelte";
	import LogoutButton from "./LogoutButton.svelte";
	import SOORevokeButton from "./SOORevokeButton.svelte";
	import { validate } from "./session";

	import { isAuthenticated } from "./store";
	// subscribe to the authentication status
	let isLoggedIn: boolean;
	isAuthenticated.subscribe((status) => {
		isLoggedIn = status;
	});

	export let messages: messageType[] = [];

	let noOfMessages: number;
	let content: messageType;
	let filter: string;
	let isDevOn: boolean;
	let conn: WebSocket;

	// truncate the number of messages show in the app
	$: if (messages.length > noOfMessages) {
		messages = messages.slice(1);
	}

	$: if(isDevOn){
		test();
	} else{
		dial(conn);
	}
	
	let filterPorts: number[] = [];

	function defineMessages (event:any) {
		let chosenValue = event.target.value;
		let presentMessages = messages.length;

		if(chosenValue<=0){
			return;
		}

		if(chosenValue < presentMessages){
			messages = messages.slice((messages.length-chosenValue),messages.length);
		}

		noOfMessages = event.target.value;
	}

	function toggleMode (event: any){
		let chosenMode = event.target.id;

		if(chosenMode=="dev"){
			isDevOn = true;
		}

		else if(chosenMode == 'prod'){
			isDevOn = false;
			if(conn!=null){
				conn.close();
			}
		}
	}

	function addMessage(message: messageType) {
		if (
			message.dstPort === null ||
			!filterPorts.includes(message.dstPort)
		) {
			messages.push(message);
			messages = messages;
		}
	}

	function filterMessages() {
		filterPorts = filter.split("&&").map(Number);
		messages = messages.filter(
			(message) => !filterPorts.includes(message.dstPort)
		);
	}

	function dial(conn: WebSocket) {
		try {
			if(!isDevOn)
			conn = new WebSocket(`wss://${location.host}/subscribe`);
		} catch (e) {
			conn = new WebSocket(`wss://${location.host}/subscribe`);
		}
		if(conn){
			conn.addEventListener("close", (ev) => {
				if (ev.code !== 1001) {
					//appendLog("Reconnecting in 1s");
					setTimeout(dial, 1000);
				}
			});
			conn.addEventListener("open", () => {
				console.info("websocket connected");
			});
			conn.addEventListener("message", (ev) => {
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

	function configButtonHandler(){
		var element = document.getElementById('configmodal');
		var backdrop = document.getElementById('backdrop');

		if(element.style.visibility == 'hidden'){
			element.style.visibility = 'visible';
			backdrop.style.visibility = 'visible';
		}
		else{
			element.style.visibility = 'hidden';
			backdrop.style.visibility = 'hidden';
		}
	}

	function submitHandler(){
		let backdrop = document.getElementById('backdrop');
		let element = document.getElementById('configmodal');
		backdrop.style.visibility = 'hidden';
		element.style.visibility = 'hidden';
	}

	const sleep = (ms: number) => new Promise((f) => setTimeout(f, ms));

	const test = async () => {
		while(isDevOn){	
			await sleep(1000);
			addMessage({
				action: "action",
				connKey: [2, 2],
				dstPort: 1234,
				rule: "Rule: TCP",
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
		// Default value of number of messages
		noOfMessages = 50;
		isDevOn = false;
		conn = null;
		validate();
	});
</script>

<header class="site-header">
	<b>Ochi</b>: find me at
	<a target="_blank" href="https://github.com/glaslos/ochi"
		>github/glaslos/ochi</a
	>
	<input bind:value={filter} placeholder="Filter destination port" />
	<button on:click={filterMessages}>Apply</button>
	<span>Port number and '&&' to concat.</span>
	<backdrop id="backdrop" on:click={submitHandler} style="visibility: hidden;"></backdrop>
	<button id="configButton" on:click={configButtonHandler}>Config</button>
		<div id="configmodal" style="visibility: hidden;">
			<p>Number of messages</p>
			<input type="number" min="0" on:change={defineMessages}>
			<p>Mode</p>
			<label>
				<input type="radio" name="radio-group" id="dev" on:click={toggleMode}>
				Development
			</label>
			<label>
				<input type="radio" name="radio-group" checked id="prod" on:click={toggleMode}>
				Production
			</label>
			<button on:click={submitHandler}>Apply</button>
		</div>
	{#if !isLoggedIn}
		<SSOButton />
	{:else}
		<LogoutButton />
		<SOORevokeButton />
	{/if}
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

    #configmodal{
        position: absolute;
		z-index: 50;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		border: 1px black solid;
		padding: 15px;
		border-radius: 10px;
		background-color: white;
    }

	#backdrop {
		position: fixed;
		top: 0;
		left: 0;
		width: 100%;
		height: 100vh;
		z-index: 20;
		background-color: rgba(0, 0, 0, 0.75);
  	}
</style>
