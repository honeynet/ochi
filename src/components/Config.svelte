<script lang="ts">
    import Button from './Button.svelte';
    import Modal from './Modal.svelte';
    import { maxNumberOfMessages, env } from '../store';

    let configModal: Modal;

    function openConfig() {
        configModal.showModal();
    }

    // function updateConfig() {
    //     if ($maxNumberOfMessages <= 0) {
    //         return;
    //     }

    //     if ($maxNumberOfMessages < messages.length) {
    //         messages = messages.slice(messages.length - $maxNumberOfMessages, messages.length);
    //     }

    //     if ($env == ENV_PROD) {
    //         if (conn != null) {
    //             conn.close();
    //         }
    //         messages = [];
    //     }
    // }
</script>

<section class="config">
    <Modal bind:this={configModal} />
    <Button onClick={openConfig} text="Config" buttonId="configButton" />
</section>

<style>
    .config {
        margin: 10px auto;
    }
</style>
