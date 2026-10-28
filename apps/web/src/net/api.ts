import type {Preferences} from '../guest/history';
export type Profile={displayName:string;avatar:Preferences['avatar'];settings:Pick<Preferences,'audio'|'controls'|'reducedMotion'>};
export type User = {
    id: string;
    username: string;
    displayName: string;
    avatar: string;
};
export type AuthResult = {
    user: User;
    recoveryCode: string;
};
async function csrf(signal?:AbortSignal) {
    const response = await fetch('/api/v1/auth/csrf', { credentials: 'same-origin', cache: 'no-store',signal });
    if (!response.ok)
        throw new Error('Layanan akun belum tersedia.');
    return (await response.json()).token as string;
}
async function post<T>(path: string, data: unknown,signal?:AbortSignal): Promise<T> {
    const token=await csrf(signal);
    // Session-owned requests may become obsolete while the CSRF lookup is pending.
    signal?.throwIfAborted();
    const response = await fetch(path, { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token }, body: JSON.stringify(data),signal });
    if (response.status === 204)
        return undefined as T;
    const body = await response.json().catch(() => ({ error: 'Layanan akun belum tersedia.' }));
    if (!response.ok)
        throw new Error(body.error || 'Layanan akun belum tersedia.');
    return body;
}
async function get<T>(path: string,signal?:AbortSignal): Promise<T> { const response = await fetch(path, { credentials: 'same-origin', cache: 'no-store',signal }); const body = await response.json().catch(() => ({ error: 'Layanan dunia belum tersedia.' })); if (!response.ok)
    throw new Error(body.error || 'Layanan dunia belum tersedia.'); return body; }
export type World = {
    id: string;
    name: string;
    ownerId: string;
    archived: boolean;
};
export type Member = {
    userId: string;
    username: string;
    role: 'owner' | 'moderator' | 'member';
};
export type Invite = {
    id: string;
    targetUsername?: string;
    expiresAt: string;
    revoked: boolean;
    redeemed: boolean;
};
export type Friend = {
    id: string;
    username: string;
    requestId: string;
    incoming: boolean;
    accepted: boolean;
    online: boolean;
};
export type Notification = {
    id: string;
    worldName: string;
    issuerUsername: string;
    expiresAt: string;
};
export type ChatMessage = {
    id: string;
    worldId: string;
    userId: string;
    username: string;
    clientMessageId: string;
    text: string;
    createdAt: string;
    deleted: boolean;
};
export const api = {
    profile:(signal?:AbortSignal)=>get<Profile>('/api/v1/account/profile',signal),
    updateMotion:(reducedMotion:boolean,signal?:AbortSignal)=>post<Profile>('/api/v1/account/motion',{reducedMotion},signal),
    updateProfile:(profile:Profile,signal?:AbortSignal)=>post<User>('/api/v1/account/profile',profile,signal),
    changePassword:(password:string,newPassword:string,signal?:AbortSignal)=>post<void>('/api/v1/account/password',{password,newPassword},signal),
    rotateRecovery:(password:string,signal?:AbortSignal)=>post<{recoveryCode:string}>('/api/v1/account/recovery',{password},signal),
    deleteAccount:(confirmation:string,signal?:AbortSignal)=>post<void>('/api/v1/account/delete',{confirmation},signal),
    friends: () => get<Friend[]>('/api/v1/friends'),
    searchFriend: (username: string) => get<Friend>(`/api/v1/friends/search?username=${encodeURIComponent(username)}`),
    requestFriend: (username: string) => post<void>('/api/v1/friends/request', { username }),
    respondFriend: (id: string, accept: boolean) => post<void>(`/api/v1/friends/${encodeURIComponent(id)}/respond`, { accept }),
    friendAction: (id: string, action: 'unfriend' | 'block' | 'unblock') => post<void>(`/api/v1/friends/${encodeURIComponent(id)}/${action}`, {}),
    blocks: () => get<Friend[]>('/api/v1/blocks'),
    notifications: () => get<Notification[]>('/api/v1/notifications'),
    acceptNotification: (id: string) => post<{
        worldId: string;
    }>(`/api/v1/notifications/${encodeURIComponent(id)}/accept`, {}),
    chatHistory: (world: string, before = '') => get<ChatMessage[]>(`/api/v1/worlds/${encodeURIComponent(world)}/chat${before ? '?before=' + encodeURIComponent(before) : ''}`),
    moderate: (world: string, action: {
        kind: 'mute' | 'delete';
        userId?: string;
        messageId?: string;
        seconds?: number;
    }) => post<void>(`/api/v1/worlds/${encodeURIComponent(world)}/moderation`, action),
    register: (username: string, password: string) => post<AuthResult>('/api/v1/auth/register', { username, password }),
    login: (username: string, password: string) => post<{
        user: User;
    }>('/api/v1/auth/login', { username, password }),
    logout: () => post<void>('/api/v1/auth/logout', {}),
    recover: (username: string, code: string, newPassword: string) => post<{
        recoveryCode: string;
    }>('/api/v1/auth/recover', { username, code, newPassword }),
    async me(): Promise<User | null> { const response = await fetch('/api/v1/me', { credentials: 'same-origin', cache: 'no-store' }); if (response.status === 401)
        return null; if (!response.ok)
        throw new Error('Layanan akun belum tersedia.'); return response.json(); },
    worlds: () => get<World[]>('/api/v1/worlds'),
    world: (id: string) => get<World>(`/api/v1/worlds/${encodeURIComponent(id)}`),
    members: (id: string) => get<Member[]>(`/api/v1/worlds/${encodeURIComponent(id)}/members`),
    bans: (id: string) => get<Member[]>(`/api/v1/worlds/${encodeURIComponent(id)}/bans`),
    invites: (id: string) => get<Invite[]>(`/api/v1/worlds/${encodeURIComponent(id)}/invites`),
    createWorld: (name: string) => post<World>('/api/v1/worlds', { name }),
    issueInvite: (id: string, username: string) => post<{
        id: string;
        token: string;
        expiresAt: string;
    }>(`/api/v1/worlds/${encodeURIComponent(id)}/invites`, { username }),
    redeemInvite: (token: string) => post<{
        worldId: string;
    }>('/api/v1/invites/redeem', { token }),
    revokeInvite: (world: string, id: string) => post<void>(`/api/v1/worlds/${encodeURIComponent(world)}/invites/${encodeURIComponent(id)}/revoke`, {}),
    changeRole: (world: string, user: string, role: string) => post<void>(`/api/v1/worlds/${encodeURIComponent(world)}/members/${encodeURIComponent(user)}/role`, { role }),
    memberAction: (world: string, user: string, action: 'kick' | 'ban' | 'unban') => post<void>(`/api/v1/worlds/${encodeURIComponent(world)}/members/${encodeURIComponent(user)}/${action}`, {}),
    transferWorld: (world: string, userId: string) => post<void>(`/api/v1/worlds/${encodeURIComponent(world)}/transfer`, { userId }),
    archiveWorld: (world: string) => post<void>(`/api/v1/worlds/${encodeURIComponent(world)}/archive`, {})
};
