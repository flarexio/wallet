import { PublicKey, Transaction, TransactionVersion, VersionedTransaction } from "@solana/web3.js";
import { BaseMessageSignerWalletAdapter, WalletName, WalletReadyState } from "@solana/wallet-adapter-base";

export interface FlareXWalletAdapterConfig {
    url?: string;
}

export const FlareXWalletName = 'FlareX' as WalletName<'FlareX'>;

export class FlareXWalletAdapter extends BaseMessageSignerWalletAdapter {
    name = FlareXWalletName;
    icon = 'https://wallet.flarex.io/favicon.ico';
    supportedTransactionVersions: ReadonlySet<TransactionVersion> = new Set(['legacy', 0]);

    private _url: string = 'https://wallet.flarex.io';
    private _connecting: boolean = false;
    private _publicKey: PublicKey | null = null;
    private _readyState: WalletReadyState = WalletReadyState.Unsupported;

    constructor(config: FlareXWalletAdapterConfig) {
        super();

        if (config.url != undefined) {
            this.url = config.url;
        }

        fetch(`${this.url}/wallet/v1/health`).then(res => {
            if (res.status == 200) {
                this._readyState = WalletReadyState.Installed;
            }
        }).catch(() => {
            this._readyState = WalletReadyState.NotDetected;
        });
    }

    async connect(): Promise<void> {
    }

    async disconnect(): Promise<void> {
    }

    async signTransaction<T extends Transaction | VersionedTransaction>(transaction: T): Promise<T> {
        return transaction;
    }

    async signMessage(message: Uint8Array): Promise<Uint8Array> {
        return new Uint8Array();
    }

    public get url(): string {
        return this._url;
    }

    public set url(url: string) {
        this._url = url;
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
}

export enum WalletMessageType {
    TRUST_SITE = 'TRUST_SITE',
    SIGN_TRANSACTION = 'SIGN_TRANSACTION',
    SIGN_MESSAGE = 'SIGN_MESSAGE',
}

export class WalletMessage {
  id: string;
  type: WalletMessageType;
  origin: string;
  payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;

  constructor(
    id: string,
    type: WalletMessageType,
    origin: string,
    payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload
  ) {
    this.id = id;
    this.type = type;
    this.origin = origin;
    this.payload = payload;
  }

  serialize(): string {
    let payload: string;
    switch (this.type) {
      case WalletMessageType.TRUST_SITE:
        const trustSitePayload = this.payload as TrustSitePayload;
        payload = JSON.stringify(trustSitePayload);
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        const signTransactionPayload = this.payload as SignTransactionPayload;
        payload = signTransactionPayload.serialize();
        break;

      case WalletMessageType.SIGN_MESSAGE:
        const signMessagePayload = this.payload as SignMessagePayload;
        payload = signMessagePayload.serialize();
        break;
    }

    return JSON.stringify({
      id: this.id,
      type: this.type,
      origin: this.origin,
      payload: payload,
    });
  }

  static deserialize(message: string): WalletMessage {
    const value = JSON.parse(message);

    let payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
    switch (value.type) {
      case WalletMessageType.TRUST_SITE:
        payload = JSON.parse(value.payload) as TrustSitePayload;
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        payload = SignTransactionPayload.deserialize(value.payload);
        break;

      case WalletMessageType.SIGN_MESSAGE:
        payload = SignMessagePayload.deserialize(value.payload);
        break;

      default:
        throw new Error(`unknown message type: ${value.type}`);
    }

    return new WalletMessage(
      value.id,
      value.type,
      value.origin,
      payload,
    );
  }
}

export class TrustSitePayload {
  app: string;
  domain: string;
  icon?: string;
  accept?: boolean;

  constructor(app: string, domain: string, icon?: string, accept?: boolean) {
    this.app = app;
    this.domain = domain;
    this.icon = icon;
    this.accept = accept;
  }
}

export class SignTransactionPayload {
  tx: Uint8Array;
  sigs?: Uint8Array[];

  constructor(tx: Uint8Array, sigs?: Uint8Array[]) {
    this.tx = tx;
    this.sigs = sigs;
  }

  serialize(): string {
    const tx = Buffer.from(this.tx).toString('base64');

    let sigs: string[] | undefined = undefined;
    if (this.sigs != undefined) {
      sigs = this.sigs.map((sig) => Buffer.from(sig).toString('base64'));
    }

    return JSON.stringify({ tx, sigs });
  }

  static deserialize(payload: string): SignTransactionPayload {
    const value = JSON.parse(payload);

    const tx = Buffer.from(value.tx, 'base64');

    let sigs: Uint8Array[] | undefined = undefined;
    if (value.sigs != undefined) {
      sigs = value.sigs.map((sig: string) => Buffer.from(sig, 'base64'));
    }

    return new SignTransactionPayload(tx, sigs);
  }
}

export class SignMessagePayload {
  msg: Uint8Array;
  sig?: Uint8Array;

  constructor(msg: Uint8Array, sig?: Uint8Array) {
    this.msg = msg;
    this.sig = sig;
  }

  serialize(): string {
    const msg = Buffer.from(this.msg).toString('base64');

    let sig: string | undefined = undefined;
    if (this.sig != undefined) {
      sig = Buffer.from(this.sig).toString('base64');
    }

    return JSON.stringify({ msg, sig });
  }

  static deserialize(payload: string): SignMessagePayload {
    const value = JSON.parse(payload);

    const msg = Buffer.from(value.msg, 'base64');

    let sig: Uint8Array | undefined = undefined;
    if (value.sig != undefined) {
      sig = Buffer.from(value.sig, 'base64');
    }

    return new SignMessagePayload(msg, sig);
  }
}

export class WalletMessageResponse {
  id: string;
  type: WalletMessageType;
  success: boolean;
  error?: string;
  payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload;

  constructor(
    id: string,
    type: WalletMessageType,
    success: boolean,
    error?: string,
    payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload
  ) {
    this.id = id;
    this.type = type;
    this.success = success;
    this.error = error;
    this.payload = payload;
  }

  serialize(): string {
    let payload: string;
    switch (this.type) {
      case WalletMessageType.TRUST_SITE:
        payload = JSON.stringify(this.payload as TrustSitePayload);
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        payload = (this.payload as SignTransactionPayload).serialize();
        break;

      case WalletMessageType.SIGN_MESSAGE:
        payload = (this.payload as SignMessagePayload).serialize();
        break;
    }

    return JSON.stringify({
      id: this.id,
      type: this.type,
      success: this.success,
      error: this.error,
      payload: payload,
    });
  }

  static deserialize(message: string): WalletMessageResponse {
    const value = JSON.parse(message);

    let payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
    switch (value.type) {
      case WalletMessageType.TRUST_SITE:
        payload = JSON.parse(value.payload) as TrustSitePayload;
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        payload = SignTransactionPayload.deserialize(value.payload);
        break;

      case WalletMessageType.SIGN_MESSAGE:
        payload = SignMessagePayload.deserialize(value.payload);
        break;

      default:
        throw new Error(`unknown message type: ${value.type}`);
    }

    return new WalletMessageResponse(
      value.id,
      value.type,
      value.success,
      value.error,
      payload,
    );
  }
}
