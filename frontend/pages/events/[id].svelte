<script lang="ts">
    import { params } from '@roxi/routify';
    import PageShell from '../../components/PageShell.svelte';
    import Content from '../../components/Content.svelte';
    import type { Event } from '../../event';
    import { API_ENDPOINT } from '../../constants';
    import { currentEvent, token } from '../../store';
    import { untrack } from 'svelte';

    async function getEventById(id: string): Promise<Event> {
        console.log('fetching event');
        const res = await fetch(`${API_ENDPOINT}/api/events/${id}`, {
            method: 'GET',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
        });

        if (res.ok) {
            console.log('received success ');
            const event = await res.json();
            return event;
        } else {
            console.log('failed to save ' + res.text());
            throw new Error('Could not fetch an event');
        }
    }

    $effect(() => {
        const id = $params.id;
        untrack(() => {
            getEventById(id).then((event) => {
                currentEvent.set(event);
            });
        });
    });
</script>

<PageShell path="/" pathText="Go back">
    <Content isShared={true} />
</PageShell>
