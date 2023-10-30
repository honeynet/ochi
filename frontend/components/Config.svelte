<script lang="ts">
    import Button from './Button.svelte';
    import Modal from './Modal.svelte';

    let configModal: Modal;

    function openConfig() {
        configModal.showModal();
    }
</script>

<section class="config">
    <Modal bind:this={configModal} />
    <Button onClick={openConfig} text="Config" buttonId="configButton" />
</section>

<style>
    .config {
        margin: 10px 20px;
    }
</style>
