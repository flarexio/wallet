import { PublicKey, Transaction, VersionedTransaction } from "@solana/web3.js";
type RetryOperation = 'Trust Site' | 'Sign Message' | 'Sign Transaction' | undefined;
export declare class FlarexWallet {
    private origin;
    private handlers;
    private walletWindow;
    private pendingMessage;
    private _requestRetry;
    constructor(origin?: string);
    private messageHandler;
    openWindow(): void;
    retryOperation(): void;
    getPublicKey(): Promise<PublicKey>;
    signMessage(message: Uint8Array): Promise<Uint8Array>;
    signTransaction<T extends Transaction | VersionedTransaction>(tx: T): Promise<T>;
    get requestRetry(): RetryOperation;
}
export {};
