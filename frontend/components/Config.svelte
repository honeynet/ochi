<script lang="ts">
    import Button from './Button.svelte';
    import Modal from './Modal.svelte';

    let configModal = $state<{ showModal: () => void } | null>(null);

    function openConfig() {
        configModal?.showModal();
    }
</script>

<Modal bind:this={configModal} />
<Button onClick={openConfig} text="Config" buttonId="configButton" />
