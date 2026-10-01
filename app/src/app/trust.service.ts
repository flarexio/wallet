import { Injectable } from '@angular/core';

import { PublicKey } from '@solana/web3.js';

const STORAGE_KEY = 'trusted-sites';

export interface TrustedSite {
  origin: string;
  app: string;
  icon?: string;
  trustedAt: number;
  lastUsedAt: number;
}

type TrustStore = { [wallet: string]: { [origin: string]: TrustedSite } };

// Grants are scoped to a wallet address, not to the browser: switching
// accounts must not inherit the previous one's sites.
@Injectable({
  providedIn: 'root'
})
export class TrustService {

  // Anything but http(s) is refused rather than normalized into a key that
  // looks legitimate.
  normalize(origin: string | undefined | null): string | null {
    if (!origin) return null;

    try {
      const url = new URL(origin);
      if (url.protocol != 'https:' && url.protocol != 'http:') return null;

      return url.origin;
    } catch {
      return null;
    }
  }

  isTrusted(wallet: PublicKey | string, origin: string): boolean {
    const key = this.normalize(origin);
    if (key == null) return false;

    return this.sites(wallet)[key] != undefined;
  }

  list(wallet: PublicKey | string | null): TrustedSite[] {
    if (wallet == null) return [];

    return Object.values(this.sites(wallet))
      .sort((a, b) => b.lastUsedAt - a.lastUsedAt);
  }

  trust(wallet: PublicKey | string, origin: string, app: string, icon?: string) {
    const key = this.normalize(origin);
    if (key == null) return;

    const sites = this.sites(wallet);
    const now = Date.now();

    sites[key] = {
      origin: key,
      app: app || key,
      icon: icon,
      trustedAt: sites[key]?.trustedAt ?? now,
      lastUsedAt: now,
    };

    this.save(wallet, sites);
  }

  touch(wallet: PublicKey | string, origin: string) {
    const key = this.normalize(origin);
    if (key == null) return;

    const sites = this.sites(wallet);
    const site = sites[key];
    if (site == undefined) return;

    site.lastUsedAt = Date.now();
    this.save(wallet, sites);
  }

  revoke(wallet: PublicKey | string, origin: string) {
    const key = this.normalize(origin);
    if (key == null) return;

    const sites = this.sites(wallet);
    if (sites[key] == undefined) return;

    delete sites[key];
    this.save(wallet, sites);
  }

  revokeAll(wallet: PublicKey | string) {
    this.save(wallet, {});
  }

  private sites(wallet: PublicKey | string): { [origin: string]: TrustedSite } {
    return this.store()[this.address(wallet)] ?? {};
  }

  private save(wallet: PublicKey | string, sites: { [origin: string]: TrustedSite }) {
    const store = this.store();
    store[this.address(wallet)] = sites;

    localStorage.setItem(STORAGE_KEY, JSON.stringify(store));
  }

  private store(): TrustStore {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw == null) return {};

    try {
      const value = JSON.parse(raw);
      if (value == null || typeof value != 'object') return {};

      return value as TrustStore;
    } catch {
      return {};
    }
  }

  private address(wallet: PublicKey | string): string {
    return wallet instanceof PublicKey ? wallet.toBase58() : wallet;
  }
}
