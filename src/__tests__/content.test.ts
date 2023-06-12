import { render } from '@testing-library/svelte';
import Content from '../Content.svelte';
import { describe, expect, test } from '@jest/globals';

describe('Content', () => {
    test('renders download button', () => {
        const { getByText } = render(Content, { props: { content: { payload: 'test' } } });
        const node = getByText('Download');
        expect(node).not.toBeNull();
    });
});
