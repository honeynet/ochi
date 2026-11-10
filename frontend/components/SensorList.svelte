<script lang="ts">
    import { untrack } from 'svelte';
    import { token, permissions } from '../store';
    import { PERM_ADD_SENSOR_BY_ID } from '../user';
    import { showError } from '../toast';
    import ConfirmDialog from './ConfirmDialog.svelte';
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
    let newId = $state('');
    let canAddById = $derived($permissions.includes(PERM_ADD_SENSOR_BY_ID));
    let adding = $state(false);
    let addError = $state('');

    // Deleting a sensor cannot be undone, so it goes through a confirmation dialog.
    let confirmDialog = $state<ConfirmDialog | null>(null);
    let sensorToDelete = $state<Sensor | null>(null);
    let deleting = $state(false);

    function askToDelete(sensor: Sensor) {
        sensorToDelete = sensor;
        confirmDialog?.showModal();
    }

    function closeConfirm() {
        confirmDialog?.close();
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
            const id = canAddById ? newId.trim() : '';
            const sensor = await addSensor($token, {
                ...(id ? { id } : {}),
                name: newName.trim(),
            });
            sensors = [...sensors, sensor];
            newName = '';
            newId = '';
        } catch (e) {
            addError = e instanceof Error ? e.message : String(e);
            showError(e);
        } finally {
            adding = false;
        }
    }

    async function loadSensors(authToken: string) {
        if (!authToken) {
            sensors = [];
            loading = false;
            return;
        }
        loading = true;
        error = '';
        try {
            const data = await getSensors(authToken);
            if (authToken !== $token) {
                return;
            }
            sensors = data;
        } catch (e) {
            if (authToken !== $token) {
                return;
            }
            error = e instanceof Error ? e.message : String(e);
            showError(e);
        } finally {
            if (authToken === $token) {
                loading = false;
            }
        }
    }

    $effect(() => {
        const currentToken = $token;
        untrack(() => {
            loadSensors(currentToken);
        });
    });
</script>

<h2 class="sensorList__title">Sensors</h2>
<form class="sensorList__add" onsubmit={handleAdd}>
    <input type="text" placeholder="Sensor name" bind:value={newName} disabled={adding} />
    {#if canAddById}
        <input
            type="text"
            placeholder="Sensor ID (optional)"
            bind:value={newId}
            disabled={adding}
        />
    {/if}
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

<ConfirmDialog
    bind:this={confirmDialog}
    confirmText={deleting ? 'Deleting...' : 'Delete'}
    busy={deleting}
    onconfirm={confirmDelete}
    onclose={() => {
        sensorToDelete = null;
    }}
>
    <p>
        Delete <strong>{sensorToDelete?.name || 'this sensor'}</strong>? Its events are kept, but
        the download link for this sensor will stop working.
    </p>
</ConfirmDialog>

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
</style>
