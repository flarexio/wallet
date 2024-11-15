import { PublicKey, Transaction, TransactionVersion, VersionedTransaction } from "@solana/web3.js";
import { BaseMessageSignerWalletAdapter, WalletName, WalletReadyState } from "@solana/wallet-adapter-base";
export interface FlareXWalletAdapterConfig {
    url?: string;
}
export declare const FlareXWalletName: WalletName<"FlareX">;
export declare class FlareXWalletAdapter extends BaseMessageSignerWalletAdapter {
    name: WalletName<"FlareX">;
    icon: string;
    supportedTransactionVersions: ReadonlySet<TransactionVersion>;
    private _url;
    private _connecting;
    private _publicKey;
    private _readyState;
    constructor(config: FlareXWalletAdapterConfig);
    connect(): Promise<void>;
    disconnect(): Promise<void>;
    signTransaction<T extends Transaction | VersionedTransaction>(transaction: T): Promise<T>;
    signMessage(message: Uint8Array): Promise<Uint8Array>;
    get url(): string;
    set url(url: string);
    get connecting(): boolean;
    get publicKey(): PublicKey | null;
    get readyState(): WalletReadyState;
}
export declare enum WalletMessageType {
    TRUST_SITE = "TRUST_SITE",
    SIGN_TRANSACTION = "SIGN_TRANSACTION",
    SIGN_MESSAGE = "SIGN_MESSAGE"
}
export interface WalletMessage {
    id: string;
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
