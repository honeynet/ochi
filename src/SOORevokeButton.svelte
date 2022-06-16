<script lang="ts">
	import { isAuthenticated, user, token } from "./store";

	let email: string
	user.subscribe(value => {
		email = value["email"];
	});

	function revokeSSO() {
		console.log(email)
		google.accounts.id.revoke(email, (done) => {
			isAuthenticated.set(false)
			user.set({})
			token.set("")
		});
	}
</script>

<button id="revokeButton" on:click={revokeSSO}>Revoke</button>

<style>
	#revokeButton {
		float: right;
	}
</style>