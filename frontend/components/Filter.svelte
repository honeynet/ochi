<script lang="ts">
    import Button from './Button.svelte';
    import QueryModal from './QueryModal.svelte';
    import Suggestion from './Suggestion.svelte';
    import { parseDSL, type FilterState } from '../dsl';
    import { debounce } from '../util';
    import {
        parsedFilter,
        stringFilter,
        filterActive,
        isAuthenticated,
        activeFilterId,
    } from '../store';
    import type { Query } from '../query';
    import { untrack } from 'svelte';

    let filter = $state('');
    let filterValid = $state(false);
    let saveModal = $state<{ showModal: (query: Query) => void } | null>(null);
    let hideSuggestions = $state(true);
    let filterState = $state<FilterState>({
        suggestions: [],
        partialToken: null,
    });
    let suggestionsDiv = $state<HTMLDivElement | undefined>(undefined);
    let inputField = $state<HTMLInputElement | undefined>(undefined);

    $effect(() => {
        const value = $stringFilter;
        untrack(() => {
            filter = value;
            applyFilter();
        });
    });

    function _filterChangeHandler() {
        // TODO: validate queries as user types them.
        if (filter === '') {
            filterValid = true;
            filterState = {
                suggestions: [],
                partialToken: null,
            };
            return;
        }
        let parseResult = parseDSL(filter, filterState);
        filterState = { ...filterState }; // force reactivity to update suggestions
        if (parseResult.lexErrors.length > 0 || parseResult.parseErrors.length > 0) {
            console.log('Found some errors', parseResult.lexErrors, parseResult.parseErrors);
            filterValid = false;
            // TODO: highlight in red
        } else {
            filterValid = true;
        }
        if (hideSuggestions) hideSuggestions = false;
    }

    const debouncedFilterChange = debounce(() => {
        _filterChangeHandler();
    }, 500);

    function applyFilter() {
        console.log(`Going to parse ${filter}`);
        filterValid = true;

        if (!filter) {
            filterActive.set(false);
            parsedFilter.set(undefined);
            stringFilter.set('');
            activeFilterId.set(undefined);
            return;
        }

        let parseResult = parseDSL(filter, filterState);
        if (parseResult.lexErrors.length > 0) {
            console.error(parseResult.lexErrors);
            filterActive.set(false);
            filterValid = false;
            return;
        }
        if (parseResult.parseErrors.length > 0) {
            console.error(parseResult.parseErrors);
            filterActive.set(false);
            filterValid = false;
            return;
        }
        parsedFilter.set(parseResult.cst);
    }

    function openSaveQuery() {
        saveModal?.showModal({
            content: filter,
        });
    }

    function handleClickOutsideSuggestionBox(event: MouseEvent) {
        const target = event.target;
        if (!(target instanceof Node)) {
            return;
        }
        if (
            suggestionsDiv &&
            !suggestionsDiv.contains(target) &&
            inputField &&
            !inputField.contains(target)
        ) {
            hideSuggestions = true;
        }
    }
</script>

<svelte:window onclick={handleClickOutsideSuggestionBox} />

<section class="filter">
    <div style="position: relative;">
        <input
            class:input-error={filter && !filterValid}
            class="filter-input"
            bind:value={filter}
            bind:this={inputField}
            placeholder="Filter destination port"
            oninput={debouncedFilterChange}
            onfocus={() => {
                hideSuggestions = false;
            }}
        />
        <Suggestion
            suggestions={filterState.suggestions}
            bind:suggestionsDiv
            hide={hideSuggestions}
            onSelect={(value) => {
                if (filterState.partialToken) {
                    filter = filter.slice(0, filterState.partialToken.startOffset - 1) + value;
                } else {
                    filter += ' ' + value;
                }
                inputField?.focus();
                _filterChangeHandler();
            }}
        />
    </div>
    <Button disabled={!filterValid} onClick={applyFilter} text="Apply" />
    {#if $isAuthenticated}<QueryModal bind:this={saveModal} />
        <Button disabled={!filterValid} onClick={openSaveQuery} text="Save" />
    {/if}
</section>

<style>
    .filter {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .filter-input {
        width: min(360px, 40vw);
        min-width: 160px;
    }

    input.input-error {
        border: 1px solid #ff0000;
    }

    input.input-error:focus {
        outline: 1px solid #ff0000;
        color: #ff0000;
    }
</style>
