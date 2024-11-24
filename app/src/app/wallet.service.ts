import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable, catchError, concatMap, map, of } from 'rxjs';

import { 
  WalletMessage, WalletMessageType, WalletMessageResponse,
  TrustSitePayload, SignTransactionPayload, SignMessagePayload,
} from '@flarex/wallet-adapter';
import { CredentialRequestOptionsJSON, get } from "@github/webauthn-json";
import { PublicKey, Transaction, VersionedTransaction } from '@solana/web3.js';
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

  private _walletSubject = new BehaviorSubject<PublicKey | null>(null);

  walletChange = this._walletSubject.asObservable();

  constructor(
    private http: HttpClient,
    private identity: IdentityService,
  ) {
    this.identity.userChange.pipe(
      concatMap((user) => this.wallet(user)),
    ).subscribe({
      next: (wallet) => this._currentWallet = wallet,
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    });
  }

  refreshWallet() {
    const wallet = this._currentWallet;
    this._walletSubject.next(wallet);
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
          trustSitePayload,
        ));

      case WalletMessageType.SIGN_MESSAGE:
        const signMsgPayload = msg.payload as SignMessagePayload;
        const message = signMsgPayload.message;

        return this.signMessage(msg.id, message).pipe(
          map((result) => {
            const signature = base58.decode(result.signature);

            return new WalletMessageResponse(
              msg.id,
              msg.type,
              true,
              new SignMessagePayload(message, signature),
            );
          }),
          catchError((err) => {
            return of(new WalletMessageResponse(
              msg.id,
              msg.type,
              false,
              undefined,
              err.message,
            ));
          }),
        );

      case WalletMessageType.SIGN_TRANSACTION:
        const signTxPayload = msg.payload as SignTransactionPayload;
        const bytes = Buffer.from(signTxPayload.transaction);
        const tx = VersionedTransaction.deserialize(bytes);

        return this.signTransaction(msg.id, tx).pipe(
          map((result) => {
            const bytes = result.transaction.serialize();
            const versioned = result.versioned;
            const signatures = result.signatures.map(
              (sig) => base58.decode(sig),
            );

            return new WalletMessageResponse(
              msg.id,
              msg.type,
              true,
              new SignTransactionPayload(bytes, versioned, signatures),
            );
          }),
          catchError((err) => {
            return of(new WalletMessageResponse(
              msg.id,
              msg.type,
              false,
              undefined,
              err.message,
            ));
          }),
        );
    }
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

    const based = Buffer
      .from(msg)
      .toString('base64');

    const body = {
      user_id, 
      transaction_id: tid,
      message: based,
    };

    return this.http.post(`${this.baseURL}/accounts/${user}/message-signatures`, body, { headers },).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.put(`${this.baseURL}/accounts/${user}/message-signatures`, credential, { headers })),
      map((resp: any) => {
        const sig = resp.signature as string;

        return { message: msg, signature: sig };
      }),
    );
  }

  signTransaction(tid: string, tx: Transaction | VersionedTransaction): Observable<SignTransactionResponse> {
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

    const versioned = tx instanceof VersionedTransaction;

    const based = Buffer
      .from(tx.serialize())
      .toString('base64');

    const body = {
      user_id, 
      transaction_id: tid,
      transaction: based,
      versioned,
    };

    return this.http.post(`${this.baseURL}/accounts/${user}/transaction-signatures`, body, { headers },).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.put(`${this.baseURL}/accounts/${user}/transaction-signatures`, credential, { headers })),
      map((resp: any) => {
        const bytes = Buffer.from(resp.transaction, 'base64');
        const versioned = resp.versioned as boolean;
        const signatures = resp.signatures as string[];

        let transaction: Transaction | VersionedTransaction;
        if (versioned) {
          transaction = VersionedTransaction.deserialize(bytes);
        } else {
          transaction = Transaction.from(bytes);
        }

        return { transaction, versioned, signatures };
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

    this._walletSubject.next(wallet);
  }
}

export interface SignMessageResponse {
  message: Uint8Array;
  signature: string;
}

export interface SignTransactionResponse {
  transaction: Transaction | VersionedTransaction;
  versioned: boolean;
  signatures: string[];
}
