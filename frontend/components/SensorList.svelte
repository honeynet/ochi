<script lang="ts">
    import { onMount } from 'svelte';
    import { token } from '../store';
    import { showError } from '../toast';
    import {
        type Sensor,
        addSensor,
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
            </li>
        {/each}
    </ul>
{/if}

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
