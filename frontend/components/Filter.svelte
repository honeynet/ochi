<script lang="ts">
    import Button from './Button.svelte';
    import QueryModal from './QueryModal.svelte';
    import Suggestion from './Suggestion.svelte';
    import { parseDSL } from '../dsl';
    import { debounce } from '../util';
    import {
        parsedFilter,
        stringFilter,
        filterActive,
        isAuthenticated,
        activeFilterId,
    } from '../store';

    let filter: string = '';
    let filterValid: boolean = false;
    let saveModal: QueryModal;
    let hideSuggestions: boolean = true;
    let filterState = {
        suggestions: [],
        partialToken: null,
    };
    let suggestionsDiv;
    let inputField;

    stringFilter.subscribe((value) => {
        filter = value;
        applyFilter();
    });

    function _filterChangeHandler() {
        // TODO: validate queries as user types them.
        if (filter == '') {
            filterValid = true;
            filterState = {
                suggestions: [],
                partialToken: null,
            };
            return;
        }
        let parseResult = parseDSL(filter, filterState);
        filterState = filterState; // force reactivity to update suggestions
        if (parseResult.lexErrors.length > 0 || parseResult.parseErrors.length > 0) {
            console.log('Found some errors', parseResult.lexErrors, parseResult.parseErrors);
            filterValid = false;
            // TODO: highlight in red
        } else {
            filterValid = true;
        }
        if (hideSuggestions) hideSuggestions = false;
    }

    function filterChangeHandler(): () => void {
        return debounce(() => {
            _filterChangeHandler();
        }, 500);
    }

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
        saveModal.showModal({
            content: filter,
        });
    }

    function handleClickOutsideSuggestionBox(event) {
        if (suggestionsDiv && !suggestionsDiv.contains(event.target) && inputField && !inputField.contains(event.target)) {
            hideSuggestions = true;
        }
    }
</script>

<svelte:window on:click={handleClickOutsideSuggestionBox} />

<section class="filter">
    <div style="position: relative;">
        <input
            class:input-error={filter && !filterValid}
            class="filter-input"
            bind:value={filter}
            bind:this={inputField}
            placeholder="Filter destination port"
            on:input={filterChangeHandler()}
            on:focus={() => { hideSuggestions = false; }}
            
        />
        <Suggestion
            suggestions={filterState.suggestions}
            bind:suggestionsDiv
            hide={hideSuggestions}
            onSelect={(value) => {
                console.log('Selected', value);
                if (filterState.partialToken) {
                    filter = filter.slice(0, filterState.partialToken.startOffset-1) + value;
                } else {
                    filter += ' ' + value;
                }
                inputField.focus();
                _filterChangeHandler();
            }} />
    </div>
    <Button disabled={!filterValid} onClick={applyFilter} text="Apply" />
    {#if $isAuthenticated}<QueryModal bind:this={saveModal} />
        <Button disabled={!filterValid} onClick={openSaveQuery} text="Save" />
    {/if}
</section>

<style>
    .filter {
        margin: 10px 30px 10px;
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 10px;
    }

    input.input-error {
        border: 1px solid #ff0000;
    }

    input.input-error:focus {
        outline: 1px solid #ff0000;
        color: #ff0000;
    }
</style>
