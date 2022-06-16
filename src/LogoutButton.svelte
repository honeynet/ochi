<script lang="ts">
	import { isAuthenticated, user, token } from "./store";

	function logout() {
		isAuthenticated.set(false);
		user.set({});
		token.set("");
	}
</script>

<button id="logoutButton" on:click={logout}>Logout</button>

<style>
	#logoutButton {
		float: right;
		margin-left: 0.5em;
	}
</style>
