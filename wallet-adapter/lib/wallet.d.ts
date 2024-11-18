import { PublicKey, VersionedTransaction } from "@solana/web3.js";
export declare class FlarexWallet {
    private origin;
    private messageCallbacks;
    private walletWindow;
    private todo;
    constructor(origin?: string);
    private messageHandler;
    openWindow(): void;
    getPublicKey(): Promise<PublicKey>;
    signTransaction(tx: VersionedTransaction): Promise<VersionedTransaction>;
    signMessage(message: Uint8Array): Promise<Uint8Array>;
}
