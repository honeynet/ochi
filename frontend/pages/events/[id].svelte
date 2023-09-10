<script lang="ts">
    import { params } from '@roxi/routify';
    import Header from '../../components/Header.svelte';
    import Content from '../../components/Content.svelte';
    import type { Event } from '../../event';
    import { API_ENDPOINT } from '../../constants';
    import { currentEvent, token } from '../../store';
    import { onMount } from 'svelte';

    console.log($params);

    async function getEventById(id): Promise<Event> {
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

    onMount(() => {
        getEventById($params.id).then((event) => {
            currentEvent.set(event);
        });
    });
</script>

<Header path="/" pathText="Go back" />
<Content isShared={true} />
