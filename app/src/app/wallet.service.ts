import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, concatMap, map, of, share } from 'rxjs';

import { CredentialRequestOptionsJSON, get } from "@github/webauthn-json";
import { PublicKey, VersionedTransaction } from '@solana/web3.js';

import { environment as env } from '../environments/environment';
import { IdentityService, User } from './identity.service';
import { Transaction } from './solana.service';

@Injectable({
  providedIn: 'root'
})
export class WalletService {
  private baseURL = env.FLAREX_WALLET_BASEURL;

  private _currentWallet: PublicKey | null = null;

  walletChange: Observable<PublicKey | null>;

  constructor(
    private http: HttpClient,
    private identity: IdentityService,
  ) {
    this.walletChange = this.identity.userChange.pipe(
      concatMap((user) => this.wallet(user)),
      share(),
    );
  }

  wallet(user: User | undefined): Observable<PublicKey | null> {
    if (user == undefined) {
      return of(null);
    }

    const token = this.identity.currentToken;
    if (token == undefined) {
      return of(null);
    }

    return this.http.get(`${this.baseURL}/wallets/${user.username}`, 
      { headers: { Authorization: `Bearer ${token.token}` } }
    ).pipe(
      map((raw) => {
        const address = raw as string;
        const pubkey = new PublicKey(address);

        this.currentWallet = pubkey;
        return pubkey;
      })
    );
  }

  signTransaction(tid: string, tx: Transaction): Observable<{token: string, tx: Transaction}> {
    if (this.identity.currentUser == undefined) {
      throw new Error('user not found');
    }

    const user = this.identity.currentUser.username;

    if (this.identity.currentToken == undefined) {
      throw new Error('token not found');
    }

    const token = this.identity.currentToken.token;
    const headers = { Authorization: `Bearer ${token}` };

    const user_id = this.identity.currentPasskeyUserID;
    if (user_id == undefined) {
      throw new Error('login without using a passkey')
    }

    const base64 = Buffer
      .from(tx.transaction.serialize())
      .toString('base64');

    const body = {
      user_id, 
      transaction_id: tid,
      transaction_data: base64,
    };

    return this.http.post(`${this.baseURL}/wallets/${user}/transaction/initialize`, body, { headers },).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.post(`${this.baseURL}/wallets/${user}/transaction/finalize`, credential, { headers })),
      map((raw: any) => {
        const token = raw.token as string;
        const bytes = Buffer.from(raw.tx, 'base64');
        const vtx = VersionedTransaction.deserialize(bytes);
        const newTx = new Transaction(vtx, tx.options);
        
        return { token, tx: newTx };
      }),
    );
  }

  public get currentWallet(): PublicKey | null {
    return this._currentWallet;
  }
  public set currentWallet(wallet: PublicKey | null) {
    this._currentWallet = wallet;
  }
}
