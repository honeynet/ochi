<script lang="ts">
    interface Props {
        text?: string;
        buttonId?: string;
        onClick: () => void;
        onMouseDown?: (event: MouseEvent) => void;
        disabled?: boolean;
    }

    let {
        text = 'button',
        buttonId = 'btn',
        onClick,
        onMouseDown,
        disabled = false,
    }: Props = $props();
</script>

<button class="button" {disabled} id={buttonId} onclick={onClick} onmousedown={onMouseDown}
    >{text}</button
>

<style>
    .button {
        margin: 0;
    }
</style>
