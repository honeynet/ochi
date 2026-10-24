<script lang="ts">
    import Button from './Button.svelte';
    import QueryModal from './QueryModal.svelte';
    import Suggestion from './Suggestion.svelte';
    import { parseDSL, type FilterState } from '../dsl';
    import { debounce } from '../util';
    import { showError } from '../toast';
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
    let inputField = $state<HTMLTextAreaElement | undefined>(undefined);
    let focused = $state(false);

    $effect(() => {
        const value = $stringFilter;
        untrack(() => {
            filter = value;
            applyFilter();
        });
    });

    // The field is a one-row textarea so long filters can wrap. At rest it stays a single
    // line like the input it replaced; while focused it grows to fit what has been typed.
    function resize() {
        if (!inputField) return;
        inputField.style.height = 'auto';
        if (focused) {
            inputField.style.height = `${inputField.scrollHeight}px`;
        }
    }

    $effect(() => {
        // Re-measure whenever the text or focus changes, including text set from code:
        // a picked suggestion or a saved query loaded through stringFilter.
        void filter;
        void focused;
        resize();
    });

    function handleKeydown(event: KeyboardEvent) {
        // A filter is one expression, so Enter applies it instead of adding a newline. Leave
        // it alone while an IME is composing, where Enter confirms the composed text.
        if (event.key === 'Enter' && !event.isComposing) {
            event.preventDefault();
            hideSuggestions = true;
            applyFilter();
        }
    }

    // Keep focus in the field while a button beside it is pressed. Losing it would collapse
    // the field mid-click and slide the button out from under the pointer, so the release
    // lands elsewhere and the click never fires.
    function keepFocus(event: MouseEvent) {
        event.preventDefault();
    }

    function clearFilter() {
        filter = '';
        applyFilter();
        inputField?.focus();
    }

    function _filterChangeHandler() {
        // TODO: validate queries as user types them.
        if (filter === '') {
            filterValid = true;
            parseDSL('', filterState);
            filterState = { ...filterState }; // force reactivity to update suggestions
            if (hideSuggestions) hideSuggestions = false;
            return;
        }
        let parseResult = parseDSL(filter, filterState);
        filterState = { ...filterState }; // force reactivity to update suggestions
        if (parseResult.lexErrors.length > 0 || parseResult.parseErrors.length > 0) {
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
            showError(`Invalid filter: ${parseResult.lexErrors[0].message}`);
            filterActive.set(false);
            filterValid = false;
            return;
        }
        if (parseResult.parseErrors.length > 0) {
            showError(`Invalid filter: ${parseResult.parseErrors[0].message}`);
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
    <div class="filter-field" ontransitionend={resize}>
        <textarea
            class:input-error={filter && !filterValid}
            class="filter-input"
            rows="1"
            wrap={focused ? 'soft' : 'off'}
            bind:value={filter}
            bind:this={inputField}
            placeholder="Filter events, e.g. tcp.port == 22"
            aria-label="Filter events"
            oninput={debouncedFilterChange}
            onkeydown={handleKeydown}
            onfocus={() => {
                focused = true;
                hideSuggestions = false;
                _filterChangeHandler();
            }}
            onblur={() => {
                focused = false;
            }}></textarea>
        {#if filter}
            <button
                type="button"
                class="filter-clear"
                aria-label="Clear filter"
                title="Clear filter"
                onmousedown={(event) => {
                    // Keep focus where it is while the button is pressed. Moving it here would
                    // widen a collapsed field mid-click and slide the button out from under
                    // the pointer before the release. clearFilter focuses the field afterwards.
                    event.preventDefault();
                }}
                onclick={clearFilter}>×</button
            >
        {/if}
        <Suggestion
            suggestions={filterState.suggestions}
            bind:suggestionsDiv
            hide={hideSuggestions}
            onSelect={(value) => {
                // the debounced handler may not have run yet; recompute against the current text
                parseDSL(filter, filterState);
                if (filterState.partialToken) {
                    filter = filter.slice(0, filterState.partialToken.startOffset) + value;
                } else {
                    filter += ' ' + value;
                }
                inputField?.focus();
                _filterChangeHandler();
            }}
        />
    </div>
    <Button disabled={!filterValid} onClick={applyFilter} onMouseDown={keepFocus} text="Apply" />
    {#if $isAuthenticated}<QueryModal bind:this={saveModal} />
        <Button
            disabled={!filterValid}
            onClick={openSaveQuery}
            onMouseDown={keepFocus}
            text="Save"
        />
    {/if}
</section>

<style>
    .filter {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .filter-field {
        position: relative;
        width: min(360px, 40vw);
        min-width: 160px;
        transition: width 150ms ease-out;
    }

    /* :focus-within rather than :focus on the textarea, so clicking the clear button,
       which takes focus, does not collapse the field before the click lands. */
    .filter-field:focus-within {
        width: min(720px, 70vw);
    }

    .filter-input {
        display: block;
        width: 100%;
        margin: 0;
        padding-right: 2em;
        resize: none;
        overflow: hidden;
        line-height: 1.4;
    }

    .filter-clear {
        position: absolute;
        top: 0;
        right: 0;
        height: 100%;
        max-height: 2.2em;
        margin: 0;
        padding: 0 0.6em;
        border: none;
        background: transparent;
        color: #888;
        cursor: pointer;
    }

    .filter-clear:hover {
        color: #333;
    }

    @media (prefers-reduced-motion: reduce) {
        .filter-field {
            transition: none;
        }
    }

    .filter-input.input-error {
        border: 1px solid #ff0000;
    }

    .filter-input.input-error:focus {
        outline: 1px solid #ff0000;
        color: #ff0000;
    }
</style>
