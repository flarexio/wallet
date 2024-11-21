"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FlarexWallet = void 0;
const web3_js_1 = require("@solana/web3.js");
const uuid_1 = require("uuid");
const message_1 = require("./message");
class FlarexWallet {
    constructor(origin) {
        this.origin = 'https://wallet.flarex.io';
        this.messageCallbacks = new Map();
        this.walletWindow = null;
        this.todo = null;
        this.messageHandler = (event) => {
            var _a;
            console.log(event);
            if (event.origin != this.origin)
                return;
            // wallet is ready
            if (event.data == 'WALLET_READY') {
                if (this.todo == null)
                    return;
                (_a = this.walletWindow) === null || _a === void 0 ? void 0 : _a.postMessage(this.todo, this.origin);
                this.todo = null;
                return;
            }
            // wallet message response
            const resp = event.data;
            const callback = this.messageCallbacks.get(resp.id);
            if (callback != undefined) {
                callback(resp);
                this.messageCallbacks.delete(resp.id);
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
        setInterval(() => {
            var _a;
            if (this.todo == null)
                return;
            (_a = this.walletWindow) === null || _a === void 0 ? void 0 : _a.postMessage('IS_READY', this.origin);
        }, 1000);
    }
    getPublicKey() {
        return new Promise((resolve, reject) => {
            if (this.todo != null) {
                reject(new Error('wallet is busy'));
                return;
            }
            this.openWindow();
            // trust site
            const msg = new message_1.WalletMessage((0, uuid_1.v4)(), message_1.WalletMessageType.TRUST_SITE, window.location.origin, new message_1.TrustSitePayload('FlareX', window.location.origin));
            this.messageCallbacks.set(msg.id, (resp) => {
                if (!resp.success) {
                    reject(new Error(resp.error));
                    return;
                }
                const payload = resp.payload;
                const pubkey = payload.pubkey;
                if (pubkey == undefined) {
                    reject(new Error('no pubkey'));
                    return;
                }
                resolve(new web3_js_1.PublicKey(pubkey));
            });
            this.todo = msg;
        });
    }
    signMessage(message) {
        return new Promise((resolve, reject) => {
            if (this.todo != null) {
                reject(new Error('wallet is busy'));
                return;
            }
            this.openWindow();
            // sign message
            const msg = new message_1.WalletMessage((0, uuid_1.v4)(), message_1.WalletMessageType.SIGN_MESSAGE, window.location.origin, new message_1.SignMessagePayload(message));
            this.messageCallbacks.set(msg.id, (resp) => {
                if (!resp.success) {
                    reject(new Error(resp.error));
                    return;
                }
                const payload = resp.payload;
                const sig = payload.signature;
                if (sig == undefined) {
                    reject(new Error('no sig'));
                    return;
                }
                resolve(sig);
            });
            this.todo = msg;
        });
    }
    signTransaction(tx) {
        return new Promise((resolve, reject) => {
            if (this.todo != null) {
                reject(new Error('wallet is busy'));
                return;
            }
            this.openWindow();
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
            this.messageCallbacks.set(msg.id, (resp) => {
                if (!resp.success) {
                    reject(new Error(resp.error));
                    return;
                }
                const payload = resp.payload;
                const tx = versioned ?
                    web3_js_1.VersionedTransaction.deserialize(payload.transaction) :
                    web3_js_1.Transaction.from(payload.transaction);
                resolve(tx);
            });
            this.todo = msg;
        });
    }
}
exports.FlarexWallet = FlarexWallet;
