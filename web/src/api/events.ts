/**
 * Server-Sent Events client for /api/events.
 *
 * Built on top of the native EventSource so the browser handles reconnects
 * (with exponential retry hinted by the server's `retry:` frame) and
 * `Last-Event-ID` replay, giving us resilient streaming without a custom
 * socket state machine.
 *
 * Frontend subscribes once at boot and wires per-store handlers; Pinia stores
 * merge events into their reactive state so every widget/page reflects the
 * latest server-originated truth without polling.
 */

export type EventStatus = 'connecting' | 'open' | 'closed';

// Canonical event names — keep in sync with internal/events/hub.go Type consts.
export type AstroEventName =
  | 'probe.changed'
  | 'widget.created'
  | 'widget.changed'
  | 'widget.deleted'
  | 'board.changed'
  | 'datasource.changed'
  | 'metric.sample';

export interface ProbeChangedPayload {
  widget_id: number;
  status: 'ok' | 'down' | 'unknown';
  latency_ms: number;
  checked_at: string;
  previous?: string;
}

export interface WidgetDeletedPayload {
  id: number;
}

export interface DataSourceChangedPayload {
  id: number;
  op: 'upsert' | 'delete';
  view?: unknown;
}

export interface MetricSamplePayload {
  data_source_id: number;
  path: string;
  shape: string;
  payload: unknown;
  cached_at: number;
  dim?: string;
}

type PayloadMap = {
  'probe.changed': ProbeChangedPayload;
  'widget.created': unknown;
  'widget.changed': unknown;
  'widget.deleted': WidgetDeletedPayload;
  'board.changed': unknown;
  'datasource.changed': DataSourceChangedPayload;
  'metric.sample': MetricSamplePayload;
};

type Handler<T> = (payload: T) => void;
type StatusListener = (status: EventStatus) => void;

export interface EventsClientOptions {
  url?: string;
}

/**
 * EventsClient wraps one EventSource and multiplexes by event name.
 */
export class EventsClient {
  private url: string;
  private es: EventSource | null = null;
  private handlers = new Map<AstroEventName, Set<Handler<unknown>>>();
  private esListeners = new Map<AstroEventName, (e: MessageEvent) => void>();
  private statusListeners = new Set<StatusListener>();
  private status: EventStatus = 'closed';
  private stopped = false;

  constructor(opts: EventsClientOptions = {}) {
    this.url = opts.url ?? defaultUrl();
  }

  start(): void {
    if (this.es) return;
    this.stopped = false;
    this.connect();
  }

  stop(): void {
    this.stopped = true;
    this.es?.close();
    this.es = null;
    this.setStatus('closed');
  }

  getStatus(): EventStatus {
    return this.status;
  }

  onStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    listener(this.status);
    return () => this.statusListeners.delete(listener);
  }

  /**
   * Register a handler for a specific event. Returns a disposer.
   * If called before start(), the handler is still bound as soon as the stream opens.
   */
  on<K extends AstroEventName>(name: K, handler: Handler<PayloadMap[K]>): () => void {
    let bucket = this.handlers.get(name);
    if (!bucket) {
      bucket = new Set();
      this.handlers.set(name, bucket);
      this.attachListener(name);
    }
    bucket.add(handler as Handler<unknown>);
    return () => {
      bucket!.delete(handler as Handler<unknown>);
      if (bucket!.size === 0) {
        this.handlers.delete(name);
        this.detachListener(name);
      }
    };
  }

  private connect(): void {
    this.setStatus('connecting');
    let es: EventSource;
    try {
      es = new EventSource(this.url, { withCredentials: false });
    } catch (err) {
      console.warn('[events] EventSource construct failed', err);
      // EventSource construction failures are extremely rare; no retry here —
      // the caller can rebuild the client on demand.
      this.setStatus('closed');
      return;
    }
    this.es = es;

    es.addEventListener('open', () => this.setStatus('open'));
    es.addEventListener('error', () => {
      // EventSource will auto-retry unless readyState === CLOSED.
      if (es.readyState === EventSource.CLOSED || this.stopped) {
        this.setStatus('closed');
      } else {
        this.setStatus('connecting');
      }
    });

    // Re-install per-name listeners for any handlers registered prior to start().
    for (const name of this.handlers.keys()) {
      this.attachListener(name);
    }
  }

  private attachListener(name: AstroEventName): void {
    if (!this.es) return;
    if (this.esListeners.has(name)) return;
    const listener = (e: MessageEvent): void => {
      let payload: unknown = null;
      try {
        payload = e.data ? JSON.parse(e.data as string) : null;
      } catch (err) {
        console.warn(`[events] parse ${name} failed`, err);
        return;
      }
      const bucket = this.handlers.get(name);
      if (!bucket) return;
      for (const h of bucket) {
        try {
          h(payload);
        } catch (err) {
          console.error(`[events] handler for ${name} threw`, err);
        }
      }
    };
    this.es.addEventListener(name, listener);
    this.esListeners.set(name, listener);
  }

  private detachListener(name: AstroEventName): void {
    const listener = this.esListeners.get(name);
    if (!listener || !this.es) return;
    this.es.removeEventListener(name, listener);
    this.esListeners.delete(name);
  }

  private setStatus(next: EventStatus): void {
    if (this.status === next) return;
    this.status = next;
    for (const l of this.statusListeners) l(next);
  }
}

function defaultUrl(): string {
  if (typeof window === 'undefined') return 'http://127.0.0.1:8080/api/events';
  return `${window.location.origin}/api/events`;
}

let singleton: EventsClient | null = null;

export function getEvents(): EventsClient {
  if (!singleton) {
    singleton = new EventsClient();
    singleton.start();
  }
  return singleton;
}
