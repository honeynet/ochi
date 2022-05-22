<script lang="ts">
	import { onMount } from "svelte";
	import { dial } from "./main";

	onMount(() => {
		dial();
	});
</script>

<main>
	<div class="row">
		<div class="column" id="message-log" />
		<div class="column" id="content" />
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

	#message-log p:first-child {
		margin: 0;
	}

	#message-log > * + * {
		margin: 5px 0 0 0;
	}
</style>
