import { useEffect, useState } from 'react';
import { config } from '../config';

interface Props {
  token: string;
  tenantId: string;
  onTokenChange: (token: string) => void;
  onTenantChange: (tenantId: string) => void;
}

export function AuthTokenPanel({ token, tenantId, onTokenChange, onTenantChange }: Props) {
  const [draftToken, setDraftToken] = useState(token);
  const [draftTenant, setDraftTenant] = useState(tenantId);

  useEffect(() => setDraftToken(token), [token]);
  useEffect(() => setDraftTenant(tenantId), [tenantId]);

  return (
    <section className="panel auth-panel" aria-label="Token settings">
      <div>
        <h2>Access</h2>
        <p>Paste a JWT for protected admin, risk, and WebSocket views.</p>
      </div>
      <label>
        Tenant
        <input value={draftTenant} onChange={(event) => setDraftTenant(event.target.value)} placeholder={config.defaultTenantId} />
      </label>
      <label className="token-input">
        Token
        <input value={draftToken} onChange={(event) => setDraftToken(event.target.value)} placeholder="Bearer token value" />
      </label>
      <div className="button-row">
        <button onClick={() => {
          localStorage.setItem('tradeops.token', draftToken);
          localStorage.setItem('tradeops.tenantId', draftTenant || config.defaultTenantId);
          onTokenChange(draftToken);
          onTenantChange(draftTenant || config.defaultTenantId);
        }}>Save</button>
        <button className="secondary" onClick={() => {
          localStorage.removeItem('tradeops.token');
          setDraftToken('');
          onTokenChange('');
        }}>Clear</button>
      </div>
    </section>
  );
}
