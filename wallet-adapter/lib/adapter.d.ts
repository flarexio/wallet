import { PublicKey, Transaction, TransactionVersion, VersionedTransaction } from "@solana/web3.js";
import { BaseMessageSignerWalletAdapter, WalletName, WalletReadyState } from "@solana/wallet-adapter-base";
import { FlarexWallet } from "./wallet";
export interface FlarexWalletAdapterConfig {
    url?: string;
}
export declare const FlarexWalletName: WalletName<"FlareX">;
export declare class FlarexWalletAdapter extends BaseMessageSignerWalletAdapter {
    name: WalletName<"FlareX">;
    url: string;
    icon: string;
    supportedTransactionVersions: ReadonlySet<TransactionVersion>;
    private _wallet;
    private _connecting;
    private _publicKey;
    private _readyState;
    constructor(config?: FlarexWalletAdapterConfig);
    autoConnect(): Promise<void>;
    connect(): Promise<void>;
    disconnect(): Promise<void>;
    signTransaction<T extends Transaction | VersionedTransaction>(transaction: T): Promise<T>;
    signMessage(message: Uint8Array): Promise<Uint8Array>;
    get wallet(): FlarexWallet | null;
    get connecting(): boolean;
    get publicKey(): PublicKey | null;
    get readyState(): WalletReadyState;
    set readyState(readyState: WalletReadyState);
}
