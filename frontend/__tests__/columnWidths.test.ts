import { describe, expect, test } from '@jest/globals';
import {
    DEFAULT_COLUMN_WIDTHS,
    MIN_COLUMN_WIDTHS,
    clampColumnWidth,
    loadColumnWidths,
    saveColumnWidths,
    buildEventColsStyle,
    type EventColumnWidths,
} from '../columnWidths';

describe('Event Column Widths', () => {
    test('has expected default column widths', () => {
        expect(DEFAULT_COLUMN_WIDTHS.time).toBe(75);
        expect(DEFAULT_COLUMN_WIDTHS.sensor).toBe(85);
        expect(DEFAULT_COLUMN_WIDTHS.source).toBe(200);
        expect(DEFAULT_COLUMN_WIDTHS.port).toBe(80);
        expect(DEFAULT_COLUMN_WIDTHS.handler).toBe(80);
        expect(DEFAULT_COLUMN_WIDTHS.scanner).toBe(140);
        expect(DEFAULT_COLUMN_WIDTHS.end).toBe(180);
        expect(DEFAULT_COLUMN_WIDTHS.frames).toBe(65);
        expect(DEFAULT_COLUMN_WIDTHS.details).toBe(65);
    });

    test('clampColumnWidth enforces minimum column bounds', () => {
        expect(clampColumnWidth('time', 10)).toBe(MIN_COLUMN_WIDTHS.time);
        expect(clampColumnWidth('sensor', 10)).toBe(MIN_COLUMN_WIDTHS.sensor);
        expect(clampColumnWidth('source', 20)).toBe(MIN_COLUMN_WIDTHS.source);
        expect(clampColumnWidth('sensor', 120.4)).toBe(120);
        expect(clampColumnWidth('end', 300)).toBe(300);
    });

    test('buildEventColsStyle produces correct CSS grid string without checkbox', () => {
        const widths: EventColumnWidths = {
            time: 80,
            sensor: 100,
            source: 250,
            port: 90,
            handler: 90,
            scanner: 150,
            end: 200,
            frames: 70,
            details: 70,
        };
        const style = buildEventColsStyle(widths, false);
        expect(style).toBe('80px 100px 250px 90px 90px 150px 200px 70px 70px');
    });

    test('buildEventColsStyle produces correct CSS grid string with checkbox', () => {
        const widths: EventColumnWidths = {
            time: 80,
            sensor: 100,
            source: 250,
            port: 90,
            handler: 90,
            scanner: 150,
            end: 200,
            frames: 70,
            details: 70,
        };
        const style = buildEventColsStyle(widths, true);
        expect(style).toBe('28px 80px 100px 250px 90px 90px 150px 200px 70px 70px');
    });

    test('saveColumnWidths and loadColumnWidths round-trip with storage', () => {
        const mockStorage: Record<string, string> = {};
        const storage = {
            getItem: (key: string) => mockStorage[key] ?? null,
            setItem: (key: string, value: string) => {
                mockStorage[key] = value;
            },
        };

        const customWidths: EventColumnWidths = {
            time: 90,
            sensor: 120,
            source: 350,
            port: 95,
            handler: 85,
            scanner: 160,
            end: 220,
            frames: 75,
            details: 75,
        };

        saveColumnWidths(customWidths, storage);
        expect(mockStorage['eventColumnWidths']).toBeDefined();

        const loaded = loadColumnWidths(storage);
        expect(loaded).toEqual(customWidths);
    });

    test('loadColumnWidths falls back to defaults on empty or invalid storage', () => {
        const emptyStorage = {
            getItem: () => null,
        };
        expect(loadColumnWidths(emptyStorage)).toEqual(DEFAULT_COLUMN_WIDTHS);

        const invalidStorage = {
            getItem: () => '{ invalid-json',
        };
        expect(loadColumnWidths(invalidStorage)).toEqual(DEFAULT_COLUMN_WIDTHS);
    });

    test('loadColumnWidths clamps stored values to minimum widths', () => {
        const storageWithNarrow = {
            getItem: () => JSON.stringify({ time: 5, sensor: 5, source: 10 }),
        };
        const loaded = loadColumnWidths(storageWithNarrow);
        expect(loaded.time).toBe(MIN_COLUMN_WIDTHS.time);
        expect(loaded.sensor).toBe(MIN_COLUMN_WIDTHS.sensor);
        expect(loaded.source).toBe(MIN_COLUMN_WIDTHS.source);
    });
});
