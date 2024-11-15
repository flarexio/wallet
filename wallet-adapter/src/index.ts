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

export interface WalletMessage {
  id: string; // uuid
  type: WalletMessageType;
  origin: string;
  payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
}

export interface TrustSitePayload {
  app: string;
  domain: string;
  icon?: string;
  accept?: boolean;
}

export interface SignTransactionPayload {
  tx: Uint8Array;
  sigs?: Uint8Array[];
}

export interface SignMessagePayload {
  msg: Uint8Array;
  sig?: Uint8Array;
}

export interface WalletMessageResponse {
  id: string;
  type: WalletMessageType;
  success: boolean;
  error?: string;
  payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
}
