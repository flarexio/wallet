"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.WalletMessageResponse = exports.SignMessagePayload = exports.SignTransactionPayload = exports.TrustSitePayload = exports.WalletMessage = exports.WalletMessageType = exports.FlareXWalletAdapter = exports.FlareXWalletName = void 0;
const wallet_adapter_base_1 = require("@solana/wallet-adapter-base");
exports.FlareXWalletName = 'FlareX';
class FlareXWalletAdapter extends wallet_adapter_base_1.BaseMessageSignerWalletAdapter {
    constructor(config) {
        super();
        this.name = exports.FlareXWalletName;
        this.icon = 'https://wallet.flarex.io/favicon.ico';
        this.supportedTransactionVersions = new Set(['legacy', 0]);
        this._url = 'https://wallet.flarex.io';
        this._connecting = false;
        this._publicKey = null;
        this._readyState = wallet_adapter_base_1.WalletReadyState.Unsupported;
        if (config.url != undefined) {
            this.url = config.url;
        }
        fetch(`${this.url}/wallet/v1/health`).then(res => {
            if (res.status == 200) {
                this._readyState = wallet_adapter_base_1.WalletReadyState.Installed;
            }
        }).catch(() => {
            this._readyState = wallet_adapter_base_1.WalletReadyState.NotDetected;
        });
    }
    connect() {
        return __awaiter(this, void 0, void 0, function* () {
        });
    }
    disconnect() {
        return __awaiter(this, void 0, void 0, function* () {
        });
    }
    signTransaction(transaction) {
        return __awaiter(this, void 0, void 0, function* () {
            return transaction;
        });
    }
    signMessage(message) {
        return __awaiter(this, void 0, void 0, function* () {
            return new Uint8Array();
        });
    }
    get url() {
        return this._url;
    }
    set url(url) {
        this._url = url;
    }
    get connecting() {
        return this._connecting;
    }
    get publicKey() {
        return this._publicKey;
    }
    get readyState() {
        return this._readyState;
    }
}
exports.FlareXWalletAdapter = FlareXWalletAdapter;
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
                payload = JSON.stringify(trustSitePayload);
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
                payload = JSON.parse(value.payload);
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
    constructor(app, domain, icon, accept) {
        this.app = app;
        this.domain = domain;
        this.icon = icon;
        this.accept = accept;
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
                payload = JSON.stringify(this.payload);
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
                payload = JSON.parse(value.payload);
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
