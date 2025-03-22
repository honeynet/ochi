<script lang="ts">
    export let suggestionId: string = '';
    export let hide: boolean = true;
    export let onSelect: (value) => void;
    export let suggestions: string[] = [];
    export let suggestionsDiv;
</script>

<div id={suggestionId} class="suggestions"
    style="display: {(suggestions.length == 0 || hide) ? " none" : "block"};"
    bind:this={suggestionsDiv}>
    <ul>
    {#each suggestions as suggestion}
        <p on:click={() => onSelect(suggestion)}>{suggestion}</p>
    {/each}
    </ul>
</div>
<style>
    .suggestions {
        position: absolute;
        z-index: 1;
        background-color: #f1f1f1;
        width: 100%;
        border: 1px solid #d3d3d3;
    }

    .suggestions ul {
        list-style-type: none;
        padding: 0;
        margin: 0;
    }

    .suggestions p {
        padding-left: 10px;
        text-decoration: none;
        display: block;
        cursor: pointer;
    }
</style>
