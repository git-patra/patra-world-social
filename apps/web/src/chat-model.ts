import type { ChatMessage } from './net/api';
/** Server IDs are decimal strings, so comparisons stay exact beyond Number.MAX_SAFE_INTEGER. */
export function mergeMessages(previous: ChatMessage[], incoming: ChatMessage[], deletedIDs: ReadonlySet<string> = new Set()): ChatMessage[] {
    const byID = new Map(previous.map(m => [m.id, m]));
    for (const original of incoming) {
        const m = deletedIDs.has(original.id) ? { ...original, text: '', deleted: true } : original;
        const old = byID.get(m.id);
        byID.set(m.id, old?.deleted ? old : m);
    }
    return [...byID.values()].sort((a, b) => a.id.length - b.id.length || a.id.localeCompare(b.id)).slice(-500);
}
export function validMessage(value: unknown, world: string): value is ChatMessage {
    const m = value as ChatMessage;
    return !!m && /^\d+$/.test(m.id) && m.worldId === world && typeof m.userId === 'string' && typeof m.username === 'string' && typeof m.clientMessageId === 'string' && typeof m.text === 'string' && Array.from(m.text).length <= 500 && typeof m.deleted === 'boolean' && typeof m.createdAt === 'string';
}
