"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WalletMessageResponse = exports.SignTransactionPayload = exports.SignMessagePayload = exports.TrustSitePayload = exports.WalletMessage = exports.WalletMessageType = void 0;
var WalletMessageType;
(function (WalletMessageType) {
    WalletMessageType["TRUST_SITE"] = "TRUST_SITE";
    WalletMessageType["SIGN_TRANSACTION"] = "SIGN_TRANSACTION";
    WalletMessageType["SIGN_MESSAGE"] = "SIGN_MESSAGE";
})(WalletMessageType || (exports.WalletMessageType = WalletMessageType = {}));
class WalletMessage {
    constructor(id, type, origin, payload) {
        this.id = id;
        this.type = type;
        this.origin = origin;
        this.payload = payload;
    }
    serialize() {
        let payload;
        switch (this.type) {
            case WalletMessageType.TRUST_SITE:
                const trustSitePayload = this.payload;
                payload = trustSitePayload.serialize();
                break;
            case WalletMessageType.SIGN_MESSAGE:
                const signMessagePayload = this.payload;
                payload = signMessagePayload.serialize();
                break;
            case WalletMessageType.SIGN_TRANSACTION:
                const signTransactionPayload = this.payload;
                payload = signTransactionPayload.serialize();
                break;
        }
        return JSON.stringify({
            id: this.id,
            type: this.type,
            origin: this.origin,
            payload: payload,
        });
    }
    static deserialize(message) {
        const value = JSON.parse(message);
        let payload;
        switch (value.type) {
            case WalletMessageType.TRUST_SITE:
                payload = TrustSitePayload.deserialize(value.payload);
                break;
            case WalletMessageType.SIGN_MESSAGE:
                payload = SignMessagePayload.deserialize(value.payload);
                break;
            case WalletMessageType.SIGN_TRANSACTION:
                payload = SignTransactionPayload.deserialize(value.payload);
                break;
            default:
                throw new Error(`unknown message type: ${value.type}`);
        }
        return new WalletMessage(value.id, value.type, value.origin, payload);
    }
}
exports.WalletMessage = WalletMessage;
class TrustSitePayload {
    constructor(app, domain, icon, accept, pubkey) {
        this.app = app;
        this.domain = domain;
        this.icon = icon;
        this.accept = accept;
        this.pubkey = pubkey;
    }
    serialize() {
        let pubkey = undefined;
        if (this.pubkey != undefined) {
            pubkey = Buffer.from(this.pubkey).toString('base64');
        }
        return JSON.stringify({
            app: this.app,
            domain: this.domain,
            icon: this.icon,
            accept: this.accept,
            pubkey: pubkey,
        });
    }
    static deserialize(payload) {
        const value = JSON.parse(payload);
        let pubkey = undefined;
        if (value.pubkey != undefined) {
            pubkey = Buffer.from(value.pubkey, 'base64');
        }
        return new TrustSitePayload(value.app, value.domain, value.icon, value.accept, pubkey);
    }
}
exports.TrustSitePayload = TrustSitePayload;
class SignMessagePayload {
    constructor(message, signature) {
        this.message = message;
        this.signature = signature;
    }
    serialize() {
        const message = Buffer.from(this.message).toString('base64');
        let signature = undefined;
        if (this.signature != undefined) {
            signature = Buffer.from(this.signature).toString('base64');
        }
        return JSON.stringify({ message, signature });
    }
    static deserialize(payload) {
        const value = JSON.parse(payload);
        const message = Buffer.from(value.message, 'base64');
        let signature = undefined;
        if (value.signature != undefined) {
            signature = Buffer.from(value.signature, 'base64');
        }
        return new SignMessagePayload(message, signature);
    }
}
exports.SignMessagePayload = SignMessagePayload;
class SignTransactionPayload {
    constructor(transaction, versioned, signatures) {
        this.transaction = transaction;
        this.versioned = versioned;
        this.signatures = signatures;
    }
    serialize() {
        const transaction = Buffer.from(this.transaction).toString('base64');
        const versioned = this.versioned;
        let signatures = undefined;
        if (this.signatures != undefined) {
            signatures = this.signatures.map((sig) => Buffer.from(sig).toString('base64'));
        }
        return JSON.stringify({ transaction, versioned, signatures });
    }
    static deserialize(payload) {
        const value = JSON.parse(payload);
        const transaction = Buffer.from(value.transaction, 'base64');
        const versioned = value.versioned;
        let signatures = undefined;
        if (value.signatures != undefined) {
            signatures = value.signatures.map((sig) => Buffer.from(sig, 'base64'));
        }
        return new SignTransactionPayload(transaction, versioned, signatures);
    }
}
exports.SignTransactionPayload = SignTransactionPayload;
class WalletMessageResponse {
    constructor(id, type, success, payload, error) {
        this.id = id;
        this.type = type;
        this.success = success;
        this.error = error;
        this.payload = payload;
    }
    serialize() {
        let payload;
        switch (this.type) {
            case WalletMessageType.TRUST_SITE:
                payload = this.payload.serialize();
                break;
            case WalletMessageType.SIGN_MESSAGE:
                payload = this.payload.serialize();
                break;
            case WalletMessageType.SIGN_TRANSACTION:
                payload = this.payload.serialize();
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
    static deserialize(message) {
        const value = JSON.parse(message);
        let payload;
        switch (value.type) {
            case WalletMessageType.TRUST_SITE:
                payload = TrustSitePayload.deserialize(value.payload);
                break;
            case WalletMessageType.SIGN_MESSAGE:
                payload = SignMessagePayload.deserialize(value.payload);
                break;
            case WalletMessageType.SIGN_TRANSACTION:
                payload = SignTransactionPayload.deserialize(value.payload);
                break;
            default:
                throw new Error(`unknown message type: ${value.type}`);
        }
        return new WalletMessageResponse(value.id, value.type, value.success, payload, value.error);
    }
}
exports.WalletMessageResponse = WalletMessageResponse;
