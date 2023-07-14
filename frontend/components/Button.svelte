<script lang="ts">
    export let text: string = 'button';
    export let buttonId: string = 'btn';
    export let onClick: () => void;
    export let disabled: boolean = false;
</script>

<button class="button" {disabled} id={buttonId} on:click={onClick}>{text}</button>

<style>
</style>
