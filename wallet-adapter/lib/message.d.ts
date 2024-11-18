export declare enum WalletMessageType {
    TRUST_SITE = "TRUST_SITE",
    SIGN_TRANSACTION = "SIGN_TRANSACTION",
    SIGN_MESSAGE = "SIGN_MESSAGE"
}
export declare class WalletMessage {
    id: string;
    type: WalletMessageType;
    origin: string;
    payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
    constructor(id: string, type: WalletMessageType, origin: string, payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload);
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
export declare class SignTransactionPayload {
    tx: Uint8Array;
    sigs?: Uint8Array[];
    constructor(tx: Uint8Array, sigs?: Uint8Array[]);
    serialize(): string;
    static deserialize(payload: string): SignTransactionPayload;
}
export declare class SignMessagePayload {
    msg: Uint8Array;
    sig?: Uint8Array;
    constructor(msg: Uint8Array, sig?: Uint8Array);
    serialize(): string;
    static deserialize(payload: string): SignMessagePayload;
}
export declare class WalletMessageResponse {
    id: string;
    type: WalletMessageType;
    success: boolean;
    error?: string;
    payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
    constructor(id: string, type: WalletMessageType, success: boolean, error?: string, payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload);
    serialize(): string;
    static deserialize(message: string): WalletMessageResponse;
}
