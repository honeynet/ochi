<script lang="ts">
  import { login } from "./session";
  import "google.accounts";

  function button() {
    google.accounts.id.renderButton(document.getElementById("googleButton"), {
      type: "icon",
      size: "small",
    });
  }

  async function sendCredentials(data: google.accounts.id.CredentialResponse) {
    const res = await fetch("/login", {
      method: "POST",
      body: data.credential,
    });

    if (res.ok) {
      const json = await res.json();
      console.log(json);
      login(json);
    } else {
      console.log("failed to login");
    }
  }

  function handleCredentialResponse(
    response: google.accounts.id.CredentialResponse
  ) {
    if (response && response.credential) {
      sendCredentials(response);
    } else {
      console.log("invalid credential response");
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
    defer
  ></script>
</svelte:head>

<button id="googleButton">Login with Google</button>

<style>
  #googleButton {
    float: right;
  }
</style>
