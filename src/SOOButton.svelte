<script lang="ts">
    	function button() {
		google.accounts.id.renderButton(
			document.getElementById("googleButton"),
			{ type: "icon", size: "small" }
		);
	}

	async function doPost(data) {
		let result = null;
		const res = await fetch("/login", {
			method: "POST",
			body: data,
		});

		const json = await res.json();
		result = JSON.stringify(json);
		console.log(result);
	}

	function handleCredentialResponse(response) {
		if (response && response.credential) {
			doPost(response.credential);
		}
	}

	function initSSO() {
		google.accounts.id.initialize({
			client_id:
				"610036027764-0lveoeejd62j594aqab5e24o2o82r8uf.apps.googleusercontent.com",
			ux_mode: "popup",
			callback: handleCredentialResponse,
		});
		button();
	}
</script>

<svelte:head>
	<script
		src="https://accounts.google.com/gsi/client"
		on:load={initSSO}
		async
		defer></script>
</svelte:head>

<button id="googleButton">Login with Google</button>
