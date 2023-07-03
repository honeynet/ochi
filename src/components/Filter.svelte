<script lang="ts">
    import Button from './Button.svelte';
    import { parseDSL } from '../dsl';
    import { debounce } from '../util';
    import { parsedFilter } from '../store';

    let filter: string = '';
    let filterValid: boolean = false;
    let focus: boolean = false;

    function handleFocus(): () => void {
        focus = true;
    }

    function filterChangeHandler(): () => void {
        return debounce(() => {
            // TODO: validate queries as user types them.
            console.log('filter is changing');
            focus = true;
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

    function resetQuery() {
        focus = false;
    }

    function applyFilter() {
        console.log(`Going to parse ${filter}`);
        if (filter == '') {
            parsedFilter.set(undefined);
            return;
        }

        let parseResult = parseDSL(filter);
        if (parseResult.lexErrors.length > 0) {
            console.error(parseResult.lexErrors);
            return;
        }
        if (parseResult.parseErrors.length > 0) {
            console.error(parseResult.parseErrors);
            return;
        }
        parsedFilter.set(parseResult.cst);
    }
</script>

<section class="filter">
    <input
        class:input-error={filter && !filterValid}
        class:input-focus={focus}
        bind:value={filter}
        placeholder="Filter destination port"
        on:input={filterChangeHandler()}
        on:focus={handleFocus}
    />
    <Button disabled={!filterValid} onClick={applyFilter} text="Apply" />
    {#if focus}
        <Button onClick={resetQuery} text="Reset Query" />
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

    input.input-focus {
        width: 75%;
    }

    input.input-error:focus {
        outline: 1px solid #ff0000;
    }
</style>
