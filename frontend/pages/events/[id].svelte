<script lang="ts">
    import { params } from '@roxi/routify';
    import PageShell from '../../components/PageShell.svelte';
    import Content from '../../components/Content.svelte';
    import type { Event } from '../../event';
    import { API_ENDPOINT } from '../../constants';
    import { currentEvent } from '../../store';
    import { untrack } from 'svelte';
    import { showError } from '../../toast';

    async function getEventById(id: string): Promise<Event> {
        const res = await fetch(`${API_ENDPOINT}/api/events/${id}`);

        if (res.ok) {
            const event = await res.json();
            return event;
        } else {
            throw new Error('Could not fetch an event');
        }
    }

    $effect(() => {
        const id = $params.id;
        untrack(() => {
            getEventById(id).then((event) => {
                currentEvent.set(event);
            }).catch((e) => showError(e));
        });
    });
</script>

<PageShell path="/" pathText="Go back">
    <Content isShared={true} />
</PageShell>
