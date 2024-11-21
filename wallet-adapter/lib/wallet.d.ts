import { PublicKey, Transaction, VersionedTransaction } from "@solana/web3.js";
export declare class FlarexWallet {
    private origin;
    private messageCallbacks;
    private walletWindow;
    private todo;
    constructor(origin?: string);
    private messageHandler;
    openWindow(): void;
    getPublicKey(): Promise<PublicKey>;
    signMessage(message: Uint8Array): Promise<Uint8Array>;
    signTransaction<T extends Transaction | VersionedTransaction>(tx: T): Promise<T>;
}
