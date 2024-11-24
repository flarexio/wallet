import { PublicKey, Transaction, TransactionMessage, VersionedTransaction } from "@solana/web3.js";
import { v4 as uuid } from 'uuid';

import { 
  WalletMessage, WalletMessageResponse, WalletMessageType, 
  TrustSitePayload, SignTransactionPayload, SignMessagePayload, 
} from './message';

class PopupBlockedError extends Error {
  constructor() {
    super('popup window blocked');
    this.name = 'PopupBlockedError';
  }
}

type ResponseHandler = {
  resolve: (value: any) => void, 
  reject: (reason?: any) => void,
}

type RetryOperation = 'Trust Site' | 'Sign Message' | 'Sign Transaction' | undefined;

export class FlarexWallet {
  private origin = 'https://wallet.flarex.io';
  private handlers: Map<string, ResponseHandler> = new Map();
  private walletWindow: WindowProxy | null = null;

  private pendingMessage: WalletMessage | null = null;

  private _requestRetry: RetryOperation = undefined;

  constructor(origin?: string) {
    this.origin = origin ?? 'https://wallet.flarex.io';

    window.addEventListener('message', this.messageHandler);
  }

  private messageHandler = (event: MessageEvent) => {
    console.log(event);

    if (event.origin != this.origin) return;

    // wallet is ready
    if (event.data == 'WALLET_READY') {
      if (this.pendingMessage == null) return;

      this.walletWindow?.postMessage(this.pendingMessage, this.origin);
      this.pendingMessage = null;
      return;
    }

    // wallet message response
    const resp = event.data as WalletMessageResponse;

    const handler = this.handlers.get(resp.id);
    if (handler == undefined) return;

    switch (resp.type) {
      case WalletMessageType.TRUST_SITE:
        if (!resp.success) {
          handler.reject(new Error(resp.error));
          return;
        }

        const trustSitePayload = resp.payload as TrustSitePayload;
        const pubkey = trustSitePayload.pubkey;
        if (pubkey == undefined) {
          handler.reject(new Error('no public key'));
          return;
        }

        handler.resolve(new PublicKey(pubkey));
        break;

      case WalletMessageType.SIGN_MESSAGE:
        if (!resp.success) {
          handler.reject(new Error(resp.error));
          return;
        }

        const signMessagePayload = resp.payload as SignMessagePayload;
        const sig = signMessagePayload.signature;
        if (sig == undefined) {
          handler.reject(new Error('no signature'));
          return;
        }

        handler.resolve(sig);
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        if (!resp.success) {
          handler.reject(new Error(resp.error));
          return;
        }

        const signTransactionPayload = resp.payload as SignTransactionPayload;
        const tx = signTransactionPayload.versioned ? 
          VersionedTransaction.deserialize(signTransactionPayload.transaction) : 
          Transaction.from(signTransactionPayload.transaction);

        handler.resolve(tx);
        break;
    }
  }

  openWindow() {
    const width = 440;
    const height = 700;
    const left = window.screenX + window.outerWidth - 10;
    const top = window.screenY;

    this.walletWindow = window.open(this.origin, 'wallet', 
      `width=${width},height=${height},top=${top},left=${left}`);

    if (this.walletWindow == null) {
      throw new PopupBlockedError();
    }

    setInterval(() => {
      if (this.pendingMessage == null) return;

      this.walletWindow?.postMessage('IS_READY', this.origin);
    }, 1000);
  }

  retryOperation() {
    if (this._requestRetry == null) {
      throw new Error('no pending retry operation');
    }

    const msg = this.pendingMessage;
    if (msg == null) {
      throw new Error('no pending message');
    }

    const handler = this.handlers.get(msg.id);
    if (handler == null) {
      throw new Error('no handler');
    }

    try {
      this.openWindow();
    } catch (err) {
      this.pendingMessage = null;
      this.handlers.delete(msg.id);
      handler.reject(err);
    } finally {
      this._requestRetry = undefined;
    }
  }

  cancelOperation() {
    if (this._requestRetry == undefined) {
      throw new Error('no pending retry operation');
    }

    this._requestRetry = undefined;

    const msg = this.pendingMessage;
    if (msg == null) {
      throw new Error('no pending message');
    }

    this.pendingMessage = null;

    const handler = this.handlers.get(msg.id);
    if (handler == null) {
      throw new Error('no handler');
    }

    this.handlers.delete(msg.id);

    handler.reject(new Error('operation cancelled'));
  }

  getPublicKey(): Promise<PublicKey> {
    return new Promise((resolve, reject) => {
      if (this.pendingMessage != null) {
        reject(new Error('wallet is busy'));
        return;
      }

      // trust site
      const msg = new WalletMessage(
        uuid(),
        WalletMessageType.TRUST_SITE,
        window.location.origin,
        new TrustSitePayload(
          'FlareX',
          window.location.origin,
        ),
      );

      try {
        this.pendingMessage = msg;
        this.handlers.set(msg.id, { resolve, reject });

        this.openWindow();
      } catch (err) {
        if (err instanceof PopupBlockedError) {
          this._requestRetry = 'Trust Site';
        }
      }
    });
  }

  signMessage(message: Uint8Array): Promise<Uint8Array> {
    return new Promise((resolve, reject) => {
      if (this.pendingMessage != null) {
        reject(new Error('wallet is busy'));
        return;
      }

      // sign message
      const msg = new WalletMessage(
        uuid(),
        WalletMessageType.SIGN_MESSAGE,
        window.location.origin,
        new SignMessagePayload(message),
      );

      try {
        this.pendingMessage = msg;
        this.handlers.set(msg.id, { resolve, reject });

        this.openWindow();
      } catch (err) {
        if (err instanceof PopupBlockedError) {
          this._requestRetry = 'Sign Message';
        }
      }
    });
  }

  signTransaction<T extends Transaction | VersionedTransaction>(tx: T): Promise<T> {
    return new Promise((resolve, reject) => {
      if (this.pendingMessage != null) {
        reject(new Error('wallet is busy'));
        return;
      }

      // sign transaction
      let vtx: VersionedTransaction;

      const versioned = tx instanceof VersionedTransaction;
      if (versioned) {
        vtx = tx;
      } else {
        if (tx.recentBlockhash == undefined) {
          reject(new Error('no recent blockhash'));
          return;
        }

        if (tx.feePayer == undefined) {
          reject(new Error('no fee payer'));
          return;
        }

        const message = new TransactionMessage({
          payerKey: tx.feePayer,
          instructions: tx.instructions,
          recentBlockhash: tx.recentBlockhash,
        }).compileToLegacyMessage();

        vtx = new VersionedTransaction(message);
      }

      const msg = new WalletMessage(
        uuid(),
        WalletMessageType.SIGN_TRANSACTION,
        window.location.origin,
        new SignTransactionPayload(vtx.serialize(), versioned),
      );

      try {
        this.pendingMessage = msg;
        this.handlers.set(msg.id, { resolve, reject });

        this.openWindow();
      } catch (err) {
        if (err instanceof PopupBlockedError) {
          this._requestRetry = 'Sign Transaction';
        }
      }
    });
  }

  public get requestRetry(): RetryOperation {
    return this._requestRetry;
  }
}
