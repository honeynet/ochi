<script lang="ts">
	import { user } from "./store";
	import { logout } from "./session";
	import "google.accounts";

	let email: string;
	user.subscribe((value) => {
		email = value["email"];
	});

	function revokeSSO() {
		google.accounts.id.revoke(
			email,
			(response: google.accounts.id.RevocationResponse) => {
				if (response.successful) {
					logout();
				} else {
					console.log(response.error);
				}
			}
		);
	}
</script>

<button id="revokeButton" on:click={revokeSSO}>Revoke</button>

<style>
	#revokeButton {
		float: right;
	}
</style>
