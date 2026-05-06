/**
 * WebSocket JSON-RPC 2.0 client.
 *
 * - Auto-connect to ws(s)://host/ws; exponential backoff reconnect (1s..30s cap).
 * - Sends `ping` every 30s; 3 misses force close and reconnect.
 * - `call(method, params)` -> Promise; configurable timeout (default 15s).
 * - Exposes connection status for UI.
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

interface PendingCall {
  resolve: (value: unknown) => void;
  reject: (reason: RpcError | Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

export interface ClientOptions {
  url?: string;
  heartbeatIntervalMs?: number;
  callTimeoutMs?: number;
  maxBackoffMs?: number;
}

const DEFAULTS = {
  heartbeatIntervalMs: 30_000,
  callTimeoutMs: 15_000,
  maxBackoffMs: 30_000,
  initialBackoffMs: 1_000,
  missedHeartbeatLimit: 3,
};

type Listener = (status: RpcStatus) => void;

export class RpcClient {
  private url: string;
  private ws: WebSocket | null = null;
  private nextId = 1;
  private pending = new Map<number, PendingCall>();
  private status: RpcStatus = 'disconnected';
  private listeners = new Set<Listener>();

  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private missedHeartbeats = 0;

  private opts: Required<Omit<ClientOptions, 'url'>>;
  private stopped = false;

  constructor(options: ClientOptions = {}) {
    this.url = options.url ?? defaultUrl();
    this.opts = {
      heartbeatIntervalMs: options.heartbeatIntervalMs ?? DEFAULTS.heartbeatIntervalMs,
      callTimeoutMs: options.callTimeoutMs ?? DEFAULTS.callTimeoutMs,
      maxBackoffMs: options.maxBackoffMs ?? DEFAULTS.maxBackoffMs,
    };
  }

  start(): void {
    this.stopped = false;
    this.connect();
  }

  stop(): void {
    this.stopped = true;
    this.clearTimers();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
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

  call<T = unknown>(method: string, params: Record<string, unknown> = {}): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error('rpc: not connected'));
        return;
      }
      const id = this.nextId++;
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`rpc: call ${method} timed out`));
      }, this.opts.callTimeoutMs);
      this.pending.set(id, {
        resolve: (value) => resolve(value as T),
        reject,
        timer,
      });
      const payload = { jsonrpc: '2.0', id, method, params };
      try {
        this.ws.send(JSON.stringify(payload));
      } catch (err) {
        clearTimeout(timer);
        this.pending.delete(id);
        reject(err instanceof Error ? err : new Error(String(err)));
      }
    });
  }

  private connect(): void {
    if (this.stopped) {
      return;
    }
    this.setStatus('connecting');
    let ws: WebSocket;
    try {
      ws = new WebSocket(this.url);
    } catch (err) {
      console.warn('[rpc] websocket construct failed', err);
      this.scheduleReconnect();
      return;
    }
    this.ws = ws;

    ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.missedHeartbeats = 0;
      this.setStatus('connected');
      this.startHeartbeat();
    };

    ws.onmessage = (event) => {
      this.handleMessage(event.data);
    };

    ws.onerror = (event) => {
      console.warn('[rpc] websocket error', event);
    };

    ws.onclose = () => {
      this.cleanupSocket();
      this.setStatus('disconnected');
      this.scheduleReconnect();
    };
  }

  private cleanupSocket(): void {
    this.clearHeartbeat();
    for (const [id, pc] of this.pending.entries()) {
      clearTimeout(pc.timer);
      pc.reject(new Error('rpc: connection closed'));
      this.pending.delete(id);
    }
  }

  private scheduleReconnect(): void {
    if (this.stopped || this.reconnectTimer) {
      return;
    }
    const delay = Math.min(
      DEFAULTS.initialBackoffMs * 2 ** this.reconnectAttempts,
      this.opts.maxBackoffMs,
    );
    this.reconnectAttempts += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  private handleMessage(raw: unknown): void {
    if (typeof raw !== 'string') {
      return;
    }
    let resp: JsonRpcResponse;
    try {
      resp = JSON.parse(raw) as JsonRpcResponse;
    } catch (err) {
      console.warn('[rpc] parse response failed', err, raw);
      return;
    }
    if (typeof resp.id !== 'number') {
      return;
    }
    const pending = this.pending.get(resp.id);
    if (!pending) {
      return;
    }
    this.pending.delete(resp.id);
    clearTimeout(pending.timer);
    if (resp.error) {
      pending.reject(resp.error);
    } else {
      pending.resolve(resp.result);
    }
  }

  private startHeartbeat(): void {
    this.clearHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      this.call<{ pong: boolean; ts: number }>('ping')
        .then(() => {
          this.missedHeartbeats = 0;
        })
        .catch(() => {
          this.missedHeartbeats += 1;
          if (this.missedHeartbeats >= DEFAULTS.missedHeartbeatLimit) {
            console.warn('[rpc] heartbeat missed limit reached, reconnecting');
            this.missedHeartbeats = 0;
            if (this.ws) {
              this.ws.close();
            }
          }
        });
    }, this.opts.heartbeatIntervalMs);
  }

  private clearHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private clearTimers(): void {
    this.clearHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
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

function defaultUrl(): string {
  if (typeof window === 'undefined') {
    return 'ws://127.0.0.1:8080/ws';
  }
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws`;
}

let singleton: RpcClient | null = null;

export function getRpc(): RpcClient {
  if (!singleton) {
    singleton = new RpcClient();
    singleton.start();
  }
  return singleton;
}
