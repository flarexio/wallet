import { PublicKey, Transaction, VersionedTransaction } from "@solana/web3.js";
import { v4 as uuid } from 'uuid';

import { 
  WalletMessage, WalletMessageResponse, WalletMessageType, 
  TrustSitePayload, SignTransactionPayload, SignMessagePayload, 
} from './message';

type MessageCallback = (resp: WalletMessageResponse) => void;

export class FlarexWallet {
  private origin = 'https://wallet.flarex.io';
  private messageCallbacks: Map<string, MessageCallback> = new Map();
  private walletWindow: WindowProxy | null = null;
  private todo: WalletMessage | null = null;

  constructor(origin?: string) {
    this.origin = origin ?? 'https://wallet.flarex.io';

    window.addEventListener('message', this.messageHandler);
  }

  private messageHandler = (event: MessageEvent) => {
    console.log(event);

    if (event.origin != this.origin) return;

    // wallet is ready
    if (event.data == 'WALLET_READY') {
      if (this.todo == null) return;

      this.walletWindow?.postMessage(this.todo, this.origin);
      this.todo = null;
      return;
    }

    // wallet message response
    const resp = event.data as WalletMessageResponse;
    const callback = this.messageCallbacks.get(resp.id);
    if (callback != undefined) {
      callback(resp);
      this.messageCallbacks.delete(resp.id);
    }
  }

  openWindow() {
    const width = 440;
    const height = 700;
    const left = window.screenX + window.outerWidth - 10;
    const top = window.screenY;

    this.walletWindow = window.open(this.origin, 'wallet', 
      `width=${width},height=${height},top=${top},left=${left}`);

    setInterval(() => {
      if (this.todo == null) return;

      this.walletWindow?.postMessage('IS_READY', this.origin);
    }, 1000);
  }

  getPublicKey(): Promise<PublicKey> {
    return new Promise((resolve, reject) => {
      if (this.todo != null) {
        reject(new Error('wallet is busy'));
        return;
      }

      this.openWindow();

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

      this.messageCallbacks.set(msg.id, (resp: WalletMessageResponse) => {
        if (!resp.success) {
          reject(new Error(resp.error));
          return;
        }

        const payload = resp.payload as TrustSitePayload;
        const pubkey = payload.pubkey;
        if (pubkey == undefined) {
          reject(new Error('no pubkey'));
          return;
        }

        resolve(new PublicKey(pubkey));
      });

      this.todo = msg;
    });
  }

  signMessage(message: Uint8Array): Promise<Uint8Array> {
    return new Promise((resolve, reject) => {
      if (this.todo != null) {
        reject(new Error('wallet is busy'));
        return;
      }

      this.openWindow();

      // sign message
      const msg = new WalletMessage(
        uuid(),
        WalletMessageType.SIGN_MESSAGE,
        window.location.origin,
        new SignMessagePayload(message),
      );

      this.messageCallbacks.set(msg.id, (resp: WalletMessageResponse) => {
        if (!resp.success) {
          reject(new Error(resp.error));
          return;
        }

        const payload = resp.payload as SignMessagePayload;
        const sig = payload.signature;
        if (sig == undefined) {
          reject(new Error('no sig'));
          return;
        }

        resolve(sig);
      });

      this.todo = msg;
    });
  }

  signTransaction<T extends Transaction | VersionedTransaction>(tx: T): Promise<T> {
    return new Promise((resolve, reject) => {
      if (this.todo != null) {
        reject(new Error('wallet is busy'));
        return;
      }

      this.openWindow();

      // sign transaction
      const versioned = tx instanceof VersionedTransaction;

      const msg = new WalletMessage(
        uuid(),
        WalletMessageType.SIGN_TRANSACTION,
        window.location.origin,
        new SignTransactionPayload(tx.serialize(), versioned),
      );

      this.messageCallbacks.set(msg.id, (resp: WalletMessageResponse) => {
        if (!resp.success) {
          reject(new Error(resp.error));
          return;
        }

        const payload = resp.payload as SignTransactionPayload;
        const tx = versioned ? 
          VersionedTransaction.deserialize(payload.transaction) : 
          Transaction.from(payload.transaction);

        resolve(tx as T);
      });

      this.todo = msg;
    });
  }
}
