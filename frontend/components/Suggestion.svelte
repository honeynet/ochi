<script lang="ts">
    import type { Suggestion } from '../dsl';

    interface Props {
        suggestionId?: string;
        hide?: boolean;
        onSelect: (value: string) => void;
        suggestions?: Suggestion[];
        suggestionsDiv?: HTMLDivElement | undefined;
    }

    let {
        suggestionId = '',
        hide = true,
        onSelect,
        suggestions = [],
        suggestionsDiv = $bindable(undefined),
    }: Props = $props();

    function getLabel(item: Suggestion): string {
        return typeof item === 'string' ? item : (item.label ?? item.value);
    }

    function getValue(item: Suggestion): string {
        return typeof item === 'string' ? item : item.value;
    }

    function getKey(item: Suggestion, idx: number): string {
        return typeof item === 'string' ? `${item}-${idx}` : `${item.value}-${item.label ?? idx}`;
    }
</script>

<div
    id={suggestionId}
    class="suggestions"
    style="display: {suggestions.length === 0 || hide ? 'none' : 'block'};"
    bind:this={suggestionsDiv}
>
    <ul>
        {#each suggestions as suggestion, i (getKey(suggestion, i))}
            <li
                role="option"
                aria-selected="false"
                tabindex="0"
                onclick={() => onSelect(getValue(suggestion))}
                onkeydown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        onSelect(getValue(suggestion));
                    }
                }}
            >
                {getLabel(suggestion)}
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
