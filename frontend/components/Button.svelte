<script lang="ts">
    interface Props {
        text?: string;
        buttonId?: string;
        onClick: () => void;
        disabled?: boolean;
    }

    let { text = 'button', buttonId = 'btn', onClick, disabled = false }: Props = $props();
</script>

<button class="button" {disabled} id={buttonId} onclick={onClick}>{text}</button>

<style>
</style>
