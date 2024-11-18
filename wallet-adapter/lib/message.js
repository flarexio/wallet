"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WalletMessageResponse = exports.SignMessagePayload = exports.SignTransactionPayload = exports.TrustSitePayload = exports.WalletMessage = exports.WalletMessageType = void 0;
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
            case WalletMessageType.SIGN_TRANSACTION:
                const signTransactionPayload = this.payload;
                payload = signTransactionPayload.serialize();
                break;
            case WalletMessageType.SIGN_MESSAGE:
                const signMessagePayload = this.payload;
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
    static deserialize(message) {
        const value = JSON.parse(message);
        let payload;
        switch (value.type) {
            case WalletMessageType.TRUST_SITE:
                payload = TrustSitePayload.deserialize(value.payload);
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
class SignTransactionPayload {
    constructor(tx, sigs) {
        this.tx = tx;
        this.sigs = sigs;
    }
    serialize() {
        const tx = Buffer.from(this.tx).toString('base64');
        let sigs = undefined;
        if (this.sigs != undefined) {
            sigs = this.sigs.map((sig) => Buffer.from(sig).toString('base64'));
        }
        return JSON.stringify({ tx, sigs });
    }
    static deserialize(payload) {
        const value = JSON.parse(payload);
        const tx = Buffer.from(value.tx, 'base64');
        let sigs = undefined;
        if (value.sigs != undefined) {
            sigs = value.sigs.map((sig) => Buffer.from(sig, 'base64'));
        }
        return new SignTransactionPayload(tx, sigs);
    }
}
exports.SignTransactionPayload = SignTransactionPayload;
class SignMessagePayload {
    constructor(msg, sig) {
        this.msg = msg;
        this.sig = sig;
    }
    serialize() {
        const msg = Buffer.from(this.msg).toString('base64');
        let sig = undefined;
        if (this.sig != undefined) {
            sig = Buffer.from(this.sig).toString('base64');
        }
        return JSON.stringify({ msg, sig });
    }
    static deserialize(payload) {
        const value = JSON.parse(payload);
        const msg = Buffer.from(value.msg, 'base64');
        let sig = undefined;
        if (value.sig != undefined) {
            sig = Buffer.from(value.sig, 'base64');
        }
        return new SignMessagePayload(msg, sig);
    }
}
exports.SignMessagePayload = SignMessagePayload;
class WalletMessageResponse {
    constructor(id, type, success, error, payload) {
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
            case WalletMessageType.SIGN_TRANSACTION:
                payload = this.payload.serialize();
                break;
            case WalletMessageType.SIGN_MESSAGE:
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
            case WalletMessageType.SIGN_TRANSACTION:
                payload = SignTransactionPayload.deserialize(value.payload);
                break;
            case WalletMessageType.SIGN_MESSAGE:
                payload = SignMessagePayload.deserialize(value.payload);
                break;
            default:
                throw new Error(`unknown message type: ${value.type}`);
        }
        return new WalletMessageResponse(value.id, value.type, value.success, value.error, payload);
    }
}
exports.WalletMessageResponse = WalletMessageResponse;
