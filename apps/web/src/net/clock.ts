/** Server time estimate is visual only; simulation and deadlines stay on Go. */
export class ServerClock {
    private server = 0;
    private received = 0;
    observe(serverMs: number, localMs = performance.now()) { this.server = serverMs; this.received = localMs; }
    now(localMs = performance.now()) { return this.server + Math.max(0, localMs - this.received); }
}
