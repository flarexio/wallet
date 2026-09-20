import { TrustService } from './trust.service';

const WALLET = 'BPFLoaderUpgradeab1e11111111111111111111111';
const OTHER_WALLET = 'Vote111111111111111111111111111111111111111';

describe('TrustService', () => {
  let trust: TrustService;

  beforeEach(() => {
    localStorage.clear();
    trust = new TrustService();
  });

  it('remembers a granted site', () => {
    expect(trust.isTrusted(WALLET, 'https://app.example.com')).toBeFalse();

    trust.trust(WALLET, 'https://app.example.com', 'Example');

    expect(trust.isTrusted(WALLET, 'https://app.example.com')).toBeTrue();
  });

  it('keys a grant by origin, not by URL', () => {
    trust.trust(WALLET, 'https://app.example.com/some/path?q=1', 'Example');

    expect(trust.isTrusted(WALLET, 'https://app.example.com')).toBeTrue();
    expect(trust.list(WALLET)[0].origin).toEqual('https://app.example.com');
  });

  it('does not carry a grant across ports or schemes', () => {
    trust.trust(WALLET, 'https://app.example.com', 'Example');

    expect(trust.isTrusted(WALLET, 'https://app.example.com:8443')).toBeFalse();
    expect(trust.isTrusted(WALLET, 'http://app.example.com')).toBeFalse();
    expect(trust.isTrusted(WALLET, 'https://evil.example.com')).toBeFalse();
  });

  it('scopes grants to the wallet that made them', () => {
    trust.trust(WALLET, 'https://app.example.com', 'Example');

    expect(trust.isTrusted(OTHER_WALLET, 'https://app.example.com')).toBeFalse();
    expect(trust.list(OTHER_WALLET)).toEqual([]);
  });

  it('refuses origins a browsing context could not have sent', () => {
    expect(trust.normalize('javascript:alert(1)')).toBeNull();
    expect(trust.normalize('file:///etc/passwd')).toBeNull();
    expect(trust.normalize('null')).toBeNull();
    expect(trust.normalize('')).toBeNull();
    expect(trust.normalize(undefined)).toBeNull();

    trust.trust(WALLET, 'javascript:alert(1)', 'Evil');
    expect(trust.list(WALLET)).toEqual([]);
  });

  it('revokes a single site without touching the rest', () => {
    trust.trust(WALLET, 'https://a.example.com', 'A');
    trust.trust(WALLET, 'https://b.example.com', 'B');

    trust.revoke(WALLET, 'https://a.example.com');

    expect(trust.isTrusted(WALLET, 'https://a.example.com')).toBeFalse();
    expect(trust.isTrusted(WALLET, 'https://b.example.com')).toBeTrue();
  });

  it('revokes everything for one wallet only', () => {
    trust.trust(WALLET, 'https://a.example.com', 'A');
    trust.trust(OTHER_WALLET, 'https://b.example.com', 'B');

    trust.revokeAll(WALLET);

    expect(trust.list(WALLET)).toEqual([]);
    expect(trust.isTrusted(OTHER_WALLET, 'https://b.example.com')).toBeTrue();
  });

  it('keeps the original grant time when a site is re-granted', () => {
    jasmine.clock().install();
    jasmine.clock().mockDate(new Date(1_000_000));

    trust.trust(WALLET, 'https://app.example.com', 'Example');

    jasmine.clock().mockDate(new Date(2_000_000));
    trust.trust(WALLET, 'https://app.example.com', 'Example');

    const site = trust.list(WALLET)[0];
    expect(site.trustedAt).toEqual(1_000_000);
    expect(site.lastUsedAt).toEqual(2_000_000);

    jasmine.clock().uninstall();
  });

  it('orders the list by most recently used', () => {
    jasmine.clock().install();
    jasmine.clock().mockDate(new Date(1_000_000));
    trust.trust(WALLET, 'https://a.example.com', 'A');

    jasmine.clock().mockDate(new Date(2_000_000));
    trust.trust(WALLET, 'https://b.example.com', 'B');

    expect(trust.list(WALLET).map((s) => s.origin))
      .toEqual(['https://b.example.com', 'https://a.example.com']);

    jasmine.clock().mockDate(new Date(3_000_000));
    trust.touch(WALLET, 'https://a.example.com');

    expect(trust.list(WALLET).map((s) => s.origin))
      .toEqual(['https://a.example.com', 'https://b.example.com']);

    jasmine.clock().uninstall();
  });

  it('survives a corrupted store rather than throwing', () => {
    localStorage.setItem('trusted-sites', 'not json');

    expect(trust.list(WALLET)).toEqual([]);
    expect(trust.isTrusted(WALLET, 'https://app.example.com')).toBeFalse();

    trust.trust(WALLET, 'https://app.example.com', 'Example');
    expect(trust.isTrusted(WALLET, 'https://app.example.com')).toBeTrue();
  });
});
