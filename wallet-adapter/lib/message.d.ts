export declare enum WalletMessageType {
    TRUST_SITE = "TRUST_SITE",
    SIGN_TRANSACTION = "SIGN_TRANSACTION",
    SIGN_MESSAGE = "SIGN_MESSAGE"
}
export type WalletMessagePayload = TrustSitePayload | SignMessagePayload | SignTransactionPayload;
export declare class WalletMessage {
    id: string;
    type: WalletMessageType;
    origin: string;
    payload: WalletMessagePayload;
    constructor(id: string, type: WalletMessageType, origin: string, payload: WalletMessagePayload);
    serialize(): string;
    static deserialize(message: string): WalletMessage;
}
export declare class TrustSitePayload {
    app: string;
    domain: string;
    icon?: string;
    accept?: boolean;
    pubkey?: Uint8Array;
    constructor(app: string, domain: string, icon?: string, accept?: boolean, pubkey?: Uint8Array);
    serialize(): string;
    static deserialize(payload: string): TrustSitePayload;
}
export declare class SignMessagePayload {
    message: Uint8Array;
    signature?: Uint8Array;
    constructor(message: Uint8Array, signature?: Uint8Array);
    serialize(): string;
    static deserialize(payload: string): SignMessagePayload;
}
export declare class SignTransactionPayload {
    transaction: Uint8Array;
    versioned: boolean;
    signatures?: Uint8Array[];
    constructor(transaction: Uint8Array, versioned: boolean, signatures?: Uint8Array[]);
    serialize(): string;
    static deserialize(payload: string): SignTransactionPayload;
}
export declare class WalletMessageResponse {
    id: string;
    type: WalletMessageType;
    success: boolean;
    payload?: WalletMessagePayload;
    error?: string;
    constructor(id: string, type: WalletMessageType, success: boolean, payload?: WalletMessagePayload, error?: string);
    serialize(): string;
    static deserialize(message: string): WalletMessageResponse;
}
