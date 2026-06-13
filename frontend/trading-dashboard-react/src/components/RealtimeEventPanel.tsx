import { useEffect, useRef, useState } from 'react';
import { connectDashboardSocket, StreamName, WebSocketStatus } from '../realtime/websocketClient';
import { DashboardContext, RealtimeEvent } from '../types';
import { StatusBadge } from './StatusBadge';

const streams: StreamName[] = ['all', 'orders', 'alerts', 'notifications', 'audit', 'market'];

export function RealtimeEventPanel({ token }: DashboardContext) {
  const [stream, setStream] = useState<StreamName>('alerts');
  const [status, setStatus] = useState<WebSocketStatus>('DISCONNECTED');
  const [events, setEvents] = useState<RealtimeEvent[]>([]);
  const socketRef = useRef<{ disconnect: () => void } | null>(null);

  useEffect(() => () => socketRef.current?.disconnect(), []);

  const connect = () => {
    socketRef.current?.disconnect();
    socketRef.current = connectDashboardSocket({
      stream,
      token,
      onStatus: setStatus,
      onEvent: (event) => setEvents((current) => [event, ...current].slice(0, 50))
    });
  };

  return (
    <section className="panel">
      <div className="section-title">
        <div>
          <h2>WebSocket Events</h2>
          <p>Last 50 messages from the selected stream.</p>
        </div>
        <StatusBadge status={status === 'CONNECTED' ? 'HEALTHY' : status === 'ERROR' ? 'UNHEALTHY' : 'UNKNOWN'} />
      </div>
      <div className="toolbar">
        <select value={stream} onChange={(event) => setStream(event.target.value as StreamName)}>
          {streams.map((item) => <option key={item} value={item}>{item}</option>)}
        </select>
        <button onClick={connect}>Connect</button>
        <button className="secondary" onClick={() => socketRef.current?.disconnect()}>Disconnect</button>
        <button className="secondary" onClick={() => setEvents([])}>Clear</button>
      </div>
      <div className="event-list">
        {events.length === 0 && <p className="empty">No events yet.</p>}
        {events.map((event, index) => (
          <article key={`${event.correlationId || index}-${index}`} className="event-item">
            <div>
              <strong>{event.type || event.topic || 'event'}</strong>
              <span>{event.timestamp || ''}</span>
            </div>
            <p>correlationId: {event.correlationId || 'n/a'} · tenantId: {event.tenantId || 'n/a'}</p>
            <pre>{JSON.stringify(event.payload ?? event, null, 2).slice(0, 900)}</pre>
          </article>
        ))}
      </div>
    </section>
  );
}
