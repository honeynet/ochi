<script lang="ts">
    interface Props {
        suggestionId?: string;
        hide?: boolean;
        onSelect: (value: string) => void;
        suggestions?: string[];
        suggestionsDiv?: HTMLDivElement | undefined;
    }

    let {
        suggestionId = '',
        hide = true,
        onSelect,
        suggestions = [],
        suggestionsDiv = $bindable(undefined),
    }: Props = $props();
</script>

<div
    id={suggestionId}
    class="suggestions"
    style="display: {suggestions.length === 0 || hide ? 'none' : 'block'};"
    bind:this={suggestionsDiv}
>
    <ul>
        {#each suggestions as suggestion (suggestion)}
            <li
                role="option"
                aria-selected="false"
                tabindex="0"
                onclick={() => onSelect(suggestion)}
                onkeydown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        onSelect(suggestion);
                    }
                }}
            >
                {suggestion}
            </li>
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

    .suggestions li {
        padding-left: 10px;
        text-decoration: none;
        display: block;
        cursor: pointer;
    }
</style>
