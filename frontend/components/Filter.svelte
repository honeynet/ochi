<script lang="ts">
    import Button from './Button.svelte';
    import QueryModal from './QueryModal.svelte';
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

    stringFilter.subscribe((value) => {
        filter = value;
        applyFilter();
    });

    function filterChangeHandler(): () => void {
        return debounce(() => {
            // TODO: validate queries as user types them.
            if (filter == '') {
                filterValid = true;
                return;
            }
            let parseResult = parseDSL(filter);
            if (parseResult.lexErrors.length > 0 || parseResult.parseErrors.length > 0) {
                console.log('Found some errors', parseResult.lexErrors, parseResult.parseErrors);
                filterValid = false;
                // TODO: highlight in red
            } else {
                filterValid = true;
            }
        }, 1000);
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

        let parseResult = parseDSL(filter);
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
</script>

<section class="filter">
    <input
        class:input-error={filter && !filterValid}
        bind:value={filter}
        placeholder="Filter destination port"
        on:input={filterChangeHandler()}
    />
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
    }
</style>
