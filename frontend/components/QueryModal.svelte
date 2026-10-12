<script lang="ts">
    import { token, userQueries } from '../store';
    import { createQuery, getQueries, updateQuery, MAX_TAGS, MAX_TAG_LENGTH } from '../query';
    import type { Query, Tag } from '../query';

    let dialog = $state<HTMLDialogElement | null>(null);
    let queryToEdit = $state<Query>({});
    let newTagType = $state('');
    let newTagValue = $state('');

    let tags = $derived(queryToEdit.tags ?? []);
    let canAddTag = $derived(
        newTagType.trim() !== '' && newTagValue.trim() !== '' && tags.length < MAX_TAGS,
    );

    export function showModal(objectToEdit: Query) {
        if (!dialog) return;
        if (!dialog.open) {
            dialog.showModal();
        }
        // Copy the tags array too: adding or removing tags must not change the saved
        // query in the list until the edit is confirmed.
        queryToEdit = { ...objectToEdit, tags: [...(objectToEdit.tags ?? [])] };
        newTagType = '';
        newTagValue = '';
    }

    function sameTag(a: Tag, b: Tag) {
        return a.type === b.type && a.value === b.value;
    }

    function addTag() {
        if (!canAddTag) return;
        const tag = { type: newTagType.trim(), value: newTagValue.trim() };
        if (!tags.some((t) => sameTag(t, tag))) {
            queryToEdit.tags = [...tags, tag];
        }
        newTagType = '';
        newTagValue = '';
    }

    function removeTag(tag: Tag) {
        queryToEdit.tags = tags.filter((t) => !sameTag(t, tag));
    }

    function addTagOnEnter(event: KeyboardEvent) {
        if (event.key === 'Enter' && !event.isComposing) {
            event.preventDefault();
            addTag();
        }
    }

    function closeModal() {
        if (dialog?.open) {
            dialog.close();
        }
        queryToEdit = {};
    }

    async function saveQueryAndReload() {
        await createQuery(
            {
                content: queryToEdit.content,
                description: queryToEdit.description,
                active: queryToEdit.active,
                tags: queryToEdit.tags,
            },
            $token,
        );
        await reloadQueries();
    }

    async function reloadQueries() {
        const queries = await getQueries($token);
        userQueries.set(queries);
    }

    async function updateQueryAndReload() {
        await updateQuery(
            {
                id: queryToEdit.id,
                content: queryToEdit.content,
                description: queryToEdit.description,
                active: queryToEdit.active,
                tags: queryToEdit.tags,
            },
            $token,
        );
        queryToEdit = {};
        await reloadQueries();
    }

    async function confirmAndCloseModal() {
        await saveQueryAndReload();
        closeModal();
    }

    async function updateAndCloseModal() {
        await updateQueryAndReload();
        closeModal();
    }
</script>

<dialog bind:this={dialog} onclose={closeModal}>
    <form class="queryModal__form">
        <label
            >Content
            <input type="text" bind:value={queryToEdit.content} />
        </label>
        <label
            >Description
            <input type="text" bind:value={queryToEdit.description} />
        </label>
        <fieldset class="queryModal__tags">
            <legend>Tags</legend>
            {#if tags.length > 0}
                <ul class="queryModal__tag-list">
                    {#each tags as tag (JSON.stringify([tag.type, tag.value]))}
                        <li class="tag">
                            <span>{tag.type}: {tag.value}</span>
                            <button
                                type="button"
                                class="tag__remove"
                                aria-label={`Remove tag ${tag.type}: ${tag.value}`}
                                onclick={() => removeTag(tag)}>×</button
                            >
                        </li>
                    {/each}
                </ul>
            {/if}
            <div class="queryModal__tag-add">
                <input
                    type="text"
                    placeholder="type"
                    aria-label="Tag type"
                    maxlength={MAX_TAG_LENGTH}
                    bind:value={newTagType}
                    onkeydown={addTagOnEnter}
                />
                <input
                    type="text"
                    placeholder="value"
                    aria-label="Tag value"
                    maxlength={MAX_TAG_LENGTH}
                    bind:value={newTagValue}
                    onkeydown={addTagOnEnter}
                />
                <button type="button" onclick={addTag} disabled={!canAddTag}>Add tag</button>
            </div>
            {#if tags.length >= MAX_TAGS}
                <p class="queryModal__hint">A query can have at most {MAX_TAGS} tags.</p>
            {/if}
        </fieldset>
        {#if queryToEdit && queryToEdit.id}
            <button type="button" onclick={updateAndCloseModal}>Update</button>
        {:else}
            <button type="button" onclick={confirmAndCloseModal}>Confirm</button>
        {/if}

        <button type="button" onclick={closeModal}>Cancel</button>
    </form>
</dialog>

<style>
    dialog {
        max-width: 32em;
        border-radius: 0.2em;
        border: none;
        padding: 0;
    }
    dialog::backdrop {
        background: rgba(0, 0, 0, 0.3);
    }
    dialog > form {
        padding: 1em;
    }
    dialog[open] {
        animation: zoom 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    dialog[open]::backdrop {
        animation: fade 0.2s ease-out;
    }
    button {
        display: block;
    }

    .queryModal__form {
        margin: 20px 20px;
    }

    .queryModal__tags {
        margin: 0 0 0.5em;
        border: 1px solid #ccc;
        border-radius: 2px;
    }

    .queryModal__tag-list {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin: 0 0 0.5em;
        padding: 0;
        list-style: none;
    }

    .tag {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 0 0 0 8px;
        border-radius: 999px;
        background: #e8e8e8;
        font-size: 13px;
    }

    .tag__remove {
        display: inline;
        margin: 0;
        padding: 0 8px;
        border: none;
        background: transparent;
        cursor: pointer;
    }

    .queryModal__tag-add {
        display: flex;
        gap: 6px;
    }

    .queryModal__tag-add input {
        flex: 1;
        min-width: 0;
        margin: 0;
    }

    .queryModal__tag-add button {
        margin: 0;
    }

    .queryModal__hint {
        margin: 0.5em 0 0;
        font-size: 13px;
        color: #666;
    }
</style>
