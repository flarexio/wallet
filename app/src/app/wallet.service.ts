import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, catchError, concatMap, map, of, share } from 'rxjs';

import { 
  WalletMessage, WalletMessageType, WalletMessageResponse,
  TrustSitePayload, SignTransactionPayload, SignMessagePayload,
} from '@flarex/wallet-adapter';
import { CredentialRequestOptionsJSON, get } from "@github/webauthn-json";
import { PublicKey, VersionedTransaction } from '@solana/web3.js';
import * as base58 from 'bs58';

import { environment as env } from '../environments/environment';
import { IdentityService, User } from './identity.service';

@Injectable({
  providedIn: 'root'
})
export class WalletService {
  private baseURL = env.FLAREX_WALLET_BASEURL + '/wallet/v1';

  private _responseCallback: ((resp: WalletMessageResponse) => void) | undefined;
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

    return this.http.get(`${this.baseURL}/accounts/${user.username}`, 
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

  session(session: string): Observable<WalletMessage> {
    return this.http.get(`${this.baseURL}/sessions/${session}`).pipe(
      map((resp: any) => {
        const based = resp.data as string;
        const jsonStr = Buffer.from(based, 'base64').toString('utf-8');
        return WalletMessage.deserialize(jsonStr);
      }),
    );
  }

  ackSession(session: string, resp: WalletMessageResponse): Observable<string> {
    const data = resp.serialize();
    const based = Buffer.from(data).toString('base64');

    const body = {
      data: based,
    };

    return this.http.post(`${this.baseURL}/sessions/${session}/ack`, body, 
      { responseType: 'text' }
    ).pipe(
      map((resp) => resp as string),
    );
  }

  messageHandler(msg: WalletMessage): Observable<WalletMessageResponse> {
    switch (msg.type) {
      case WalletMessageType.TRUST_SITE:
        const trustSitePayload = msg.payload as TrustSitePayload;

        const account = this.currentWallet;
        if (account == null) {
          trustSitePayload.accept = false;
        } else {
          trustSitePayload.accept = true;
          trustSitePayload.pubkey = account.toBytes();
        }

        // TODO: check if the site is trusted

        return of(new WalletMessageResponse(
          msg.id,
          msg.type,
          true,
          undefined,
          trustSitePayload,
        ));

      case WalletMessageType.SIGN_TRANSACTION:
        const signTxPayload = msg.payload as SignTransactionPayload;
        const bytes = Buffer.from(signTxPayload.tx);
        const tx = VersionedTransaction.deserialize(bytes);

        return this.signTransaction(msg.id, tx).pipe(
          map((result) => {
            const bytes = result.tx.serialize();
            const sigs = result.tx.signatures;

            return new WalletMessageResponse(
              msg.id,
              msg.type,
              true,
              undefined,
              new SignTransactionPayload(bytes, sigs),
            );
          }),
          catchError((err) => {
            return of(new WalletMessageResponse(
              msg.id,
              msg.type,
              false,
              err.message,
              undefined,
            ));
          }),
        );

      case WalletMessageType.SIGN_MESSAGE:
        const signMsgPayload = msg.payload as SignMessagePayload;

        return this.signMessage(msg.id, signMsgPayload.msg).pipe(
          map((result) => {
            const sig = base58.decode(result.sig);

            return new WalletMessageResponse(
              msg.id,
              msg.type,
              true,
              undefined,
              new SignMessagePayload(signMsgPayload.msg, sig),
            );
          }),
          catchError((err) => {
            return of(new WalletMessageResponse(
              msg.id,
              msg.type,
              false,
              err.message,
              undefined,
            ));
          }),
        );
    }
  }

  signTransaction(tid: string, tx: VersionedTransaction): Observable<SignTransactionResponse> {
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
      .from(tx.serialize())
      .toString('base64');

    const body = {
      user_id, 
      transaction_id: tid,
      transaction_data: base64,
    };

    return this.http.post(`${this.baseURL}/accounts/${user}/transaction-signatures`, body, { headers },).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.put(`${this.baseURL}/accounts/${user}/transaction-signatures`, credential, { headers })),
      map((raw: any) => {
        const token = raw.token as string;
        const bytes = Buffer.from(raw.tx, 'base64');
        const vtx = VersionedTransaction.deserialize(bytes);

        return { token, tx: vtx };
      }),
    );
  }

  signMessage(tid: string, msg: Uint8Array): Observable<SignMessageResponse> {
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
      .from(msg)
      .toString('base64');

    const body = {
      user_id, 
      transaction_id: tid,
      transaction_data: base64,
    };

    return this.http.post(`${this.baseURL}/accounts/${user}/message-signatures`, body, { headers },).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.put(`${this.baseURL}/accounts/${user}/message-signatures`, credential, { headers })),
      map((raw: any) => {
        const token = raw.token as string;
        const sig = raw.sig as string;

        return { token, msg, sig };
      }),
    );
  }

  sendResponse(resp: WalletMessageResponse) {
    if (this._responseCallback) {
      this._responseCallback(resp);
    }
  }

  public set responseCallback(callback: ((resp: WalletMessageResponse) => void) | undefined) {
    this._responseCallback = callback;
  }

  public clearResponseCallback() {
    this._responseCallback = undefined;
  }

  public get currentWallet(): PublicKey | null {
    return this._currentWallet;
  }
  public set currentWallet(wallet: PublicKey | null) {
    this._currentWallet = wallet;
  }
}

export interface SignMessageResponse {
  token: string;
  msg: Uint8Array;
  sig: string;
}

export interface SignTransactionResponse {
  token: string;
  tx: VersionedTransaction;
}
