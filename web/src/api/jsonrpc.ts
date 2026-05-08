/**
 * JSON-RPC 2.0 client over HTTP (POST /api/rpc).
 *
 * Shape contract preserved from the previous WebSocket client so no caller has
 * to change: `getRpc().call(method, params)` returns a Promise<T>, and
 * `onStatus(listener)` fires with 'connecting' | 'connected' | 'disconnected'.
 *
 * Since HTTP is stateless the "status" concept is derived from the outcome of
 * recent requests, not a socket lifecycle:
 *  - start() does one cheap `ping` to confirm reachability → 'connected'
 *  - any network/server error → 'disconnected' (listeners re-fire)
 *  - any subsequent successful call → 'connected'
 *
 * This matches what existing consumers actually want — they re-hydrate when the
 * transport becomes healthy, and back off when it's not.
 */

export type RpcStatus = 'connecting' | 'connected' | 'disconnected';

export interface RpcError {
  code: number;
  message: string;
  data?: unknown;
}

export interface JsonRpcResponse<T = unknown> {
  jsonrpc: '2.0';
  id: number | string | null;
  result?: T;
  error?: RpcError;
}

export interface ClientOptions {
  url?: string;
  /** Per-call timeout (ms). Longer than typical LAN latency so slow probes don't trip it. */
  callTimeoutMs?: number;
  /** Interval for background health check pings; set <=0 to disable. */
  healthPingIntervalMs?: number;
}

type Listener = (status: RpcStatus) => void;

const DEFAULT_TIMEOUT_MS = 15_000;
const DEFAULT_HEALTH_PING_MS = 30_000;

export class RpcClient {
  private url: string;
  private nextId = 1;
  private status: RpcStatus = 'disconnected';
  private listeners = new Set<Listener>();
  private opts: Required<Omit<ClientOptions, 'url'>>;
  private healthTimer: ReturnType<typeof setInterval> | null = null;
  private stopped = false;

  constructor(options: ClientOptions = {}) {
    this.url = options.url ?? defaultUrl();
    this.opts = {
      callTimeoutMs: options.callTimeoutMs ?? DEFAULT_TIMEOUT_MS,
      healthPingIntervalMs: options.healthPingIntervalMs ?? DEFAULT_HEALTH_PING_MS,
    };
  }

  /**
   * Bootstraps the "connection" by doing one ping so consumers get a 'connected'
   * event on mount, matching the WS client's behavior. Idempotent.
   */
  start(): void {
    this.stopped = false;
    this.setStatus('connecting');
    void this.call('ping').catch(() => {
      // status transition already happened in call()
    });
    if (this.opts.healthPingIntervalMs > 0 && !this.healthTimer) {
      this.healthTimer = setInterval(() => {
        if (this.stopped) return;
        void this.call('ping').catch(() => undefined);
      }, this.opts.healthPingIntervalMs);
    }
  }

  stop(): void {
    this.stopped = true;
    if (this.healthTimer) {
      clearInterval(this.healthTimer);
      this.healthTimer = null;
    }
    this.setStatus('disconnected');
  }

  getStatus(): RpcStatus {
    return this.status;
  }

  onStatus(listener: Listener): () => void {
    this.listeners.add(listener);
    listener(this.status);
    return () => this.listeners.delete(listener);
  }

  async call<T = unknown>(method: string, params: Record<string, unknown> = {}): Promise<T> {
    const id = this.nextId++;
    const payload = { jsonrpc: '2.0', id, method, params };
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.opts.callTimeoutMs);
    let res: Response;
    try {
      res = await fetch(this.url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        credentials: 'same-origin',
        signal: controller.signal,
      });
    } catch (err) {
      clearTimeout(timer);
      this.setStatus('disconnected');
      throw networkError(method, err);
    }
    clearTimeout(timer);

    // 204 is reserved for notifications; we always send an id so this is unexpected.
    if (res.status === 204) {
      this.setStatus('connected');
      return undefined as T;
    }

    if (!res.ok) {
      // Non-2xx usually means the body is still a JSON-RPC error envelope
      // (e.g. 400 "empty body"); try to parse for a structured error first.
      let body: JsonRpcResponse<T> | undefined;
      try {
        body = (await res.json()) as JsonRpcResponse<T>;
      } catch {
        /* fall through */
      }
      if (body?.error) {
        this.setStatus('connected'); // server answered → transport healthy
        throw body.error;
      }
      this.setStatus('disconnected');
      throw Object.assign(new Error(`rpc: http ${res.status} on ${method}`), {
        code: res.status,
      });
    }

    let envelope: JsonRpcResponse<T>;
    try {
      envelope = (await res.json()) as JsonRpcResponse<T>;
    } catch (err) {
      this.setStatus('disconnected');
      throw networkError(method, err);
    }

    this.setStatus('connected');
    if (envelope.error) {
      throw envelope.error;
    }
    return envelope.result as T;
  }

  private setStatus(next: RpcStatus): void {
    if (this.status === next) {
      return;
    }
    this.status = next;
    for (const listener of this.listeners) {
      listener(next);
    }
  }
}

function networkError(method: string, err: unknown): RpcError {
  const msg = err instanceof Error ? err.message : String(err);
  return { code: 0, message: `rpc: ${method} transport failed: ${msg}` };
}

function defaultUrl(): string {
  if (typeof window === 'undefined') {
    return 'http://127.0.0.1:8080/api/rpc';
  }
  return `${window.location.origin}/api/rpc`;
}

let singleton: RpcClient | null = null;

export function getRpc(): RpcClient {
  if (!singleton) {
    singleton = new RpcClient();
    singleton.start();
  }
  return singleton;
}
