import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
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
import { TrustService } from './trust.service';
import { TrustSiteComponent, TrustSiteRequest } from './trust-site/trust-site.component';

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
    private dialog: MatDialog,
    private identity: IdentityService,
    private trust: TrustService,
  ) {
    this.identity.userChange.pipe(
      concatMap((user) => {
        console.log('User changed in wallet service:', user?.username);
        return this.wallet(user);
      }),
    ).subscribe({
      next: (wallet) => {
        console.log('Wallet updated:', wallet?.toBase58());
        this.currentWallet = wallet; // 這會觸發 _walletSubject.next()
      },
      error: (err) => console.error('Wallet service error:', err),
      complete: () => console.log('Wallet service complete'),
    });
  }

  wallet(user: User | undefined): Observable<PublicKey | null> {
    if (!user) {
      return of(null);
    }

    const token = this.identity.currentToken;
    if (!token) {
      return of(null);
    }

    return this.http.get(`${this.baseURL}/accounts/${user.username}`, 
      { headers: { Authorization: `Bearer ${token.token}` } }
    ).pipe(
      map((raw) => {
        const address = raw as string;
        return new PublicKey(address);
      }),
      catchError(error => {
        console.error('Error fetching wallet:', error);
        return of(null);
      })
    );
  }

  refreshWallet() {
    this._walletSubject.next(this._currentWallet);
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

  /**
   * Handles a request from a dApp. `verifiedOrigin` is the browser-supplied
   * origin of the window that sent it — the only origin that can be believed.
   * The session transport has none, so it passes nothing and the request is
   * treated as unverified.
   */
  messageHandler(msg: WalletMessage, verifiedOrigin?: string): Observable<WalletMessageResponse> {
    switch (msg.type) {
      case WalletMessageType.TRUST_SITE: {
        const trustSitePayload = msg.payload as TrustSitePayload;

        const account = this.currentWallet;
        if (account == null) {
          trustSitePayload.accept = false;

          return of(new WalletMessageResponse(
            msg.id,
            msg.type,
            true,
            trustSitePayload,
          ));
        }

        return this.authorize(msg, verifiedOrigin, 'connect').pipe(
          map((granted) => {
            trustSitePayload.accept = granted;
            trustSitePayload.pubkey = granted ? account.toBytes() : undefined;

            return new WalletMessageResponse(
              msg.id,
              msg.type,
              true,
              trustSitePayload,
            );
          }),
        );
      }

      case WalletMessageType.SIGN_MESSAGE: {
        const signMsgPayload = msg.payload as SignMessagePayload;
        const message = signMsgPayload.message;

        return this.authorize(msg, verifiedOrigin, 'sign message').pipe(
          concatMap((granted) => {
            if (!granted) {
              return this.refuse(msg);
            }

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

      case WalletMessageType.SIGN_TRANSACTION: {
        const signTxPayload = msg.payload as SignTransactionPayload;
        const bytes = Buffer.from(signTxPayload.transaction);
        const tx = VersionedTransaction.deserialize(bytes);

        return this.authorize(msg, verifiedOrigin, 'sign transaction').pipe(
          concatMap((granted) => {
            if (!granted) {
              return this.refuse(msg);
            }

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
  }

  /**
   * Decides whether a site may act on the current wallet, asking the user when
   * it has not been connected before.
   *
   * A stored grant is only honoured for an origin the browser vouched for: an
   * unverified request must not be able to ride on a decision the user made in
   * a window, so it is always put back in front of them, and its answer is
   * never written to the trust list.
   */
  private authorize(
    msg: WalletMessage,
    verifiedOrigin: string | undefined,
    action: TrustSiteRequest['action'],
  ): Observable<boolean> {
    const account = this.currentWallet;
    if (account == null) return of(false);

    const verified = this.trust.normalize(verifiedOrigin);
    const claimed = this.trust.normalize(msg.origin);

    const origin = verified ?? claimed;
    if (origin == null) return of(false);

    if (verified != null && this.trust.isTrusted(account, origin)) {
      this.trust.touch(account, origin);
      return of(true);
    }

    const payload = msg.payload as Partial<TrustSitePayload>;

    const req: TrustSiteRequest = {
      origin,
      app: payload.app ?? origin,
      icon: payload.icon,
      verified: verified != null,
      claimed: claimed != null && claimed != origin ? claimed : undefined,
      action,
    };

    return this.dialog.open(TrustSiteComponent, {
      data: req,
      disableClose: true,
      width: '360px',
    }).afterClosed().pipe(
      map((granted) => {
        if (!granted) return false;

        if (verified != null) {
          this.trust.trust(account, origin, req.app, req.icon);
        }

        return true;
      }),
    );
  }

  private refuse(msg: WalletMessage): Observable<WalletMessageResponse> {
    return of(new WalletMessageResponse(
      msg.id,
      msg.type,
      false,
      undefined,
      'site not trusted',
    ));
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
    console.log('Setting current wallet:', wallet?.toBase58());
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
