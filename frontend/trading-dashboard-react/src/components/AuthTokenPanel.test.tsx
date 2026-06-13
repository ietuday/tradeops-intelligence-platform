import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { AuthTokenPanel } from './AuthTokenPanel';

describe('AuthTokenPanel', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('saves and clears the token', async () => {
    const user = userEvent.setup();
    const onTokenChange = vi.fn();
    const onTenantChange = vi.fn();
    render(<AuthTokenPanel token="" tenantId="default-tenant" onTokenChange={onTokenChange} onTenantChange={onTenantChange} />);

    await user.type(screen.getByLabelText('Token'), 'demo-token');
    await user.click(screen.getByText('Save'));

    expect(localStorage.getItem('tradeops.token')).toBe('demo-token');
    expect(onTokenChange).toHaveBeenCalledWith('demo-token');

    await user.click(screen.getByText('Clear'));

    expect(localStorage.getItem('tradeops.token')).toBeNull();
    expect(onTokenChange).toHaveBeenLastCalledWith('');
  });
});
