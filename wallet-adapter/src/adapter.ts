import { PublicKey, Transaction, TransactionVersion, VersionedTransaction } from "@solana/web3.js";
import { 
  BaseMessageSignerWalletAdapter, WalletName, WalletReadyState, 
  WalletAccountError, WalletNotConnectedError, WalletNotReadyError, WalletSignMessageError, WalletSignTransactionError
} from "@solana/wallet-adapter-base";

import { FlarexWallet } from "./wallet";

export interface FlarexWalletAdapterConfig {
  url?: string;
}

export const FlarexWalletName = 'FlareX' as WalletName<'FlareX'>;

export class FlarexWalletAdapter extends BaseMessageSignerWalletAdapter {
  name = FlarexWalletName;
  url = 'https://wallet.flarex.io';
  icon = 'https://wallet.flarex.io/favicon.ico';
  supportedTransactionVersions: ReadonlySet<TransactionVersion> = new Set([0]);

  private _wallet: FlarexWallet | null = null;
  private _connecting: boolean = false;
  private _publicKey: PublicKey | null = null;
  private _readyState: WalletReadyState = WalletReadyState.NotDetected;

  constructor(config?: FlarexWalletAdapterConfig) {
    super();

    if (config != undefined) {
      if (config.url != undefined) {
        this.url = config.url;
      }
    }

    this._wallet = new FlarexWallet(this.url);

    fetch(`${this.url}/wallet/v1/health`).then(resp => {
      if (resp.status == 200) {
        this.readyState = WalletReadyState.Installed;
      } else {
        this.readyState = WalletReadyState.Unsupported;
      }
    }).catch(() => {
      this.readyState = WalletReadyState.Unsupported;
    });
  }

  async autoConnect(): Promise<void> {
    if (this.readyState != WalletReadyState.Installed) {
      return;
    }

    try {
      const pubkey = localStorage.getItem('flarex_wallet_pubkey');
      if (pubkey == null) {
        throw new WalletNotConnectedError();
      }

      const publicKey = new PublicKey(pubkey);

      this._publicKey = publicKey;
      this.emit('connect', publicKey);
    } catch (err: any) {
      this.emit('error', err);
      throw err;
    }
  }

  async connect(): Promise<void> {
    try {
      if (this.connected || this.connecting) return;

      if (this.readyState != WalletReadyState.Installed) {
        throw new WalletNotReadyError();
      }

      this._connecting = true;

      const wallet = this._wallet;
      if (wallet == null) {
        throw new WalletNotConnectedError();
      }

      try {
        const pubkey = await wallet.getPublicKey();

        localStorage.setItem('flarex_wallet_pubkey', pubkey.toBase58());

        this._publicKey = pubkey;
        this.emit('connect', pubkey);
      } catch (err: any) {
        throw new WalletAccountError(err as string);
      }

    } catch (err: any) {
      this.emit('error', err);
      throw err;

    } finally {
      this._connecting = false;
    }
  }

  async disconnect(): Promise<void> {
    localStorage.removeItem('flarex_wallet_pubkey');

    this._publicKey = null;
    this.emit('disconnect');
  }

  async signMessage(message: Uint8Array): Promise<Uint8Array> {
    try {
      const wallet = this._wallet;
      if (wallet == null) {
        throw new WalletNotConnectedError();
      }

      try {
        const sig = await wallet.signMessage(message);
        return sig;
      } catch (err: any) {
        throw new WalletSignMessageError(err as string);
      }

    } catch (err: any) {
      this.emit('error', err);
      throw err;
    }
  }

  async signTransaction<T extends Transaction | VersionedTransaction>(transaction: T): Promise<T> {
    try {
      const wallet = this._wallet;
      if (wallet == null) {
        throw new WalletNotConnectedError();
      }

      try {
        const tx = await wallet.signTransaction(transaction);
        return tx;
      } catch (err: any) {
        throw new WalletSignTransactionError(err as string);
      }

    } catch (err: any) {
      this.emit('error', err);
      throw err;
    }
  }

  public get wallet(): FlarexWallet | null {
    return this._wallet;
  }

  public get connecting(): boolean {
      return this._connecting;
  }

  public get publicKey(): PublicKey | null {
      return this._publicKey;
  }

  public get readyState(): WalletReadyState {
      return this._readyState;
  }
  public set readyState(readyState: WalletReadyState) {
    this._readyState = readyState;
    this.emit('readyStateChange', readyState);
  }
}
