"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FlarexWallet = void 0;
const web3_js_1 = require("@solana/web3.js");
const uuid_1 = require("uuid");
const message_1 = require("./message");
class PopupBlockedError extends Error {
    constructor() {
        super('popup window blocked');
        this.name = 'PopupBlockedError';
    }
}
class FlarexWallet {
    constructor(origin) {
        this.origin = 'https://wallet.flarex.io';
        this.handlers = new Map();
        this.walletWindow = null;
        this.pendingMessage = null;
        this._requestRetry = undefined;
        this.messageHandler = (event) => {
            var _a;
            console.log(event);
            if (event.origin != this.origin)
                return;
            // wallet is ready
            if (event.data == 'WALLET_READY') {
                if (this.pendingMessage == null)
                    return;
                (_a = this.walletWindow) === null || _a === void 0 ? void 0 : _a.postMessage(this.pendingMessage, this.origin);
                this.pendingMessage = null;
                return;
            }
            // wallet message response
            const resp = event.data;
            const handler = this.handlers.get(resp.id);
            if (handler == undefined)
                return;
            switch (resp.type) {
                case message_1.WalletMessageType.TRUST_SITE:
                    if (!resp.success) {
                        handler.reject(new Error(resp.error));
                        return;
                    }
                    const trustSitePayload = resp.payload;
                    const pubkey = trustSitePayload.pubkey;
                    if (pubkey == undefined) {
                        handler.reject(new Error('no public key'));
                        return;
                    }
                    handler.resolve(new web3_js_1.PublicKey(pubkey));
                    break;
                case message_1.WalletMessageType.SIGN_MESSAGE:
                    if (!resp.success) {
                        handler.reject(new Error(resp.error));
                        return;
                    }
                    const signMessagePayload = resp.payload;
                    const sig = signMessagePayload.signature;
                    if (sig == undefined) {
                        handler.reject(new Error('no signature'));
                        return;
                    }
                    handler.resolve(sig);
                    break;
                case message_1.WalletMessageType.SIGN_TRANSACTION:
                    if (!resp.success) {
                        handler.reject(new Error(resp.error));
                        return;
                    }
                    const signTransactionPayload = resp.payload;
                    const tx = signTransactionPayload.versioned ?
                        web3_js_1.VersionedTransaction.deserialize(signTransactionPayload.transaction) :
                        web3_js_1.Transaction.from(signTransactionPayload.transaction);
                    handler.resolve(tx);
                    break;
            }
        };
        this.origin = origin !== null && origin !== void 0 ? origin : 'https://wallet.flarex.io';
        window.addEventListener('message', this.messageHandler);
    }
    openWindow() {
        const width = 440;
        const height = 700;
        const left = window.screenX + window.outerWidth - 10;
        const top = window.screenY;
        this.walletWindow = window.open(this.origin, 'wallet', `width=${width},height=${height},top=${top},left=${left}`);
        if (this.walletWindow == null) {
            throw new PopupBlockedError();
        }
        setInterval(() => {
            var _a;
            if (this.pendingMessage == null)
                return;
            (_a = this.walletWindow) === null || _a === void 0 ? void 0 : _a.postMessage('IS_READY', this.origin);
        }, 1000);
    }
    retryOperation() {
        if (this._requestRetry == null) {
            throw new Error('no pending retry operation');
        }
        const msg = this.pendingMessage;
        if (msg == null) {
            throw new Error('no pending message');
        }
        const handler = this.handlers.get(msg.id);
        if (handler == null) {
            throw new Error('no handler');
        }
        try {
            this.openWindow();
        }
        catch (err) {
            this.pendingMessage = null;
            this.handlers.delete(msg.id);
            handler.reject(err);
        }
        finally {
            this._requestRetry = undefined;
        }
    }
    cancelOperation() {
        if (this._requestRetry == undefined) {
            throw new Error('no pending retry operation');
        }
        this._requestRetry = undefined;
        const msg = this.pendingMessage;
        if (msg == null) {
            throw new Error('no pending message');
        }
        this.pendingMessage = null;
        const handler = this.handlers.get(msg.id);
        if (handler == null) {
            throw new Error('no handler');
        }
        this.handlers.delete(msg.id);
        handler.reject(new Error('operation cancelled'));
    }
    getPublicKey() {
        return new Promise((resolve, reject) => {
            if (this.pendingMessage != null) {
                reject(new Error('wallet is busy'));
                return;
            }
            // trust site
            const msg = new message_1.WalletMessage((0, uuid_1.v4)(), message_1.WalletMessageType.TRUST_SITE, window.location.origin, new message_1.TrustSitePayload('FlareX', window.location.origin));
            try {
                this.pendingMessage = msg;
                this.handlers.set(msg.id, { resolve, reject });
                this.openWindow();
            }
            catch (err) {
                if (err instanceof PopupBlockedError) {
                    this._requestRetry = 'Trust Site';
                }
            }
        });
    }
    signMessage(message) {
        return new Promise((resolve, reject) => {
            if (this.pendingMessage != null) {
                reject(new Error('wallet is busy'));
                return;
            }
            // sign message
            const msg = new message_1.WalletMessage((0, uuid_1.v4)(), message_1.WalletMessageType.SIGN_MESSAGE, window.location.origin, new message_1.SignMessagePayload(message));
            try {
                this.pendingMessage = msg;
                this.handlers.set(msg.id, { resolve, reject });
                this.openWindow();
            }
            catch (err) {
                if (err instanceof PopupBlockedError) {
                    this._requestRetry = 'Sign Message';
                }
            }
        });
    }
    signTransaction(tx) {
        return new Promise((resolve, reject) => {
            if (this.pendingMessage != null) {
                reject(new Error('wallet is busy'));
                return;
            }
            // sign transaction
            let vtx;
            const versioned = tx instanceof web3_js_1.VersionedTransaction;
            if (versioned) {
                vtx = tx;
            }
            else {
                if (tx.recentBlockhash == undefined) {
                    reject(new Error('no recent blockhash'));
                    return;
                }
                if (tx.feePayer == undefined) {
                    reject(new Error('no fee payer'));
                    return;
                }
                const message = new web3_js_1.TransactionMessage({
                    payerKey: tx.feePayer,
                    instructions: tx.instructions,
                    recentBlockhash: tx.recentBlockhash,
                }).compileToLegacyMessage();
                vtx = new web3_js_1.VersionedTransaction(message);
            }
            const msg = new message_1.WalletMessage((0, uuid_1.v4)(), message_1.WalletMessageType.SIGN_TRANSACTION, window.location.origin, new message_1.SignTransactionPayload(vtx.serialize(), versioned));
            try {
                this.pendingMessage = msg;
                this.handlers.set(msg.id, { resolve, reject });
                this.openWindow();
            }
            catch (err) {
                if (err instanceof PopupBlockedError) {
                    this._requestRetry = 'Sign Transaction';
                }
            }
        });
    }
    get requestRetry() {
        return this._requestRetry;
    }
}
exports.FlarexWallet = FlarexWallet;
