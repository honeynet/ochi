<script lang="ts">
    import { onMount } from 'svelte';
    import { token } from '../store';
    import { showError } from '../toast';
    import {
        type Sensor,
        addSensor,
        deleteSensor,
        getSensors,
        sensorDownloadUrl,
        DOWNLOAD_OS,
        DOWNLOAD_ARCH,
    } from '../sensor';

    let sensors = $state<Sensor[]>([]);
    let loading = $state(true);
    let error = $state('');

    let newName = $state('');
    let adding = $state(false);
    let addError = $state('');

    // Deleting a sensor cannot be undone, so it goes through a confirmation dialog.
    let confirmDialog = $state<HTMLDialogElement | null>(null);
    let sensorToDelete = $state<Sensor | null>(null);
    let deleting = $state(false);

    function askToDelete(sensor: Sensor) {
        sensorToDelete = sensor;
        if (confirmDialog && !confirmDialog.open) {
            confirmDialog.showModal();
        }
    }

    function closeConfirm() {
        if (confirmDialog?.open) {
            confirmDialog.close();
        }
        sensorToDelete = null;
    }

    async function confirmDelete() {
        const sensor = sensorToDelete;
        if (!sensor) return;
        deleting = true;
        try {
            await deleteSensor($token, sensor.id);
            sensors = sensors.filter((s) => s.id !== sensor.id);
            closeConfirm();
        } catch (e) {
            showError(e);
        } finally {
            deleting = false;
        }
    }

    async function handleAdd(event: SubmitEvent) {
        event.preventDefault();
        adding = true;
        addError = '';
        try {
            const sensor = await addSensor($token, {
                id: crypto.randomUUID(),
                name: newName.trim(),
            });
            sensors = [...sensors, sensor];
            newName = '';
        } catch (e) {
            addError = e instanceof Error ? e.message : String(e);
            showError(e);
        } finally {
            adding = false;
        }
    }

    onMount(async () => {
        try {
            sensors = await getSensors($token);
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
            showError(e);
        } finally {
            loading = false;
        }
    });
</script>

<h2 class="sensorList__title">Sensors</h2>
<form class="sensorList__add" onsubmit={handleAdd}>
    <input type="text" placeholder="Sensor name" bind:value={newName} disabled={adding} />
    <button type="submit" disabled={adding}>Add sensor</button>
</form>
{#if addError}
    <p class="sensorList__message">{addError}</p>
{/if}
{#if error}
    <p class="sensorList__message">{error}</p>
{:else if loading}
    <p class="sensorList__message">Loading...</p>
{:else if sensors.length === 0}
    <p class="sensorList__message">No sensors yet.</p>
{:else}
    <ul class="sensorList__items">
        {#each sensors as sensor, index (sensor.id)}
            <li class="sensorList__item">
                {index + 1}.
                <span class="sensorList__name">{sensor.name || 'Unnamed sensor'}</span>
                <span class="sensorList__id">{sensor.id}</span>
                <a href={sensorDownloadUrl(sensor)} download>
                    Download ({DOWNLOAD_OS}/{DOWNLOAD_ARCH})
                </a>
                <button onclick={() => askToDelete(sensor)}>Delete</button>
            </li>
        {/each}
    </ul>
{/if}

<dialog
    bind:this={confirmDialog}
    onclick={(event) => {
        if (event.target === event.currentTarget) {
            closeConfirm();
        }
    }}
>
    <div class="sensorList__confirm">
        <p>
            Delete <strong>{sensorToDelete?.name || 'this sensor'}</strong>? Its events are kept,
            but the download link for this sensor will stop working.
        </p>
        <div class="sensorList__confirm-buttons">
            <button onclick={closeConfirm} disabled={deleting}>Cancel</button>
            <button onclick={confirmDelete} disabled={deleting}>
                {deleting ? 'Deleting...' : 'Delete'}
            </button>
        </div>
    </div>
</dialog>

<style>
    .sensorList__title,
    .sensorList__message {
        text-align: center;
    }

    .sensorList__add {
        display: flex;
        justify-content: center;
        gap: 10px;
        margin-bottom: 20px;
    }

    .sensorList__items {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 15px;
    }

    .sensorList__item {
        display: flex;
        gap: 10px;
        align-items: baseline;
        font-size: 20px;
    }

    .sensorList__id {
        font-family: monospace;
        font-size: 14px;
        font-style: italic;
    }

    dialog {
        max-width: 32em;
        border-radius: 0.2em;
        border: none;
        padding: 0;
    }

    dialog::backdrop {
        background: rgba(0, 0, 0, 0.3);
    }

    .sensorList__confirm {
        padding: 1em;
    }

    .sensorList__confirm-buttons {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
    }
</style>
