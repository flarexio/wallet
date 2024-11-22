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
exports.FlarexWalletAdapter = exports.FlarexWalletName = void 0;
const web3_js_1 = require("@solana/web3.js");
const wallet_adapter_base_1 = require("@solana/wallet-adapter-base");
const wallet_1 = require("./wallet");
exports.FlarexWalletName = 'FlareX';
class FlarexWalletAdapter extends wallet_adapter_base_1.BaseMessageSignerWalletAdapter {
    constructor(config) {
        super();
        this.name = exports.FlarexWalletName;
        this.url = 'https://wallet.flarex.io';
        this.icon = 'https://wallet.flarex.io/favicon.ico';
        this.supportedTransactionVersions = new Set([0]);
        this._wallet = null;
        this._connecting = false;
        this._publicKey = null;
        this._readyState = wallet_adapter_base_1.WalletReadyState.NotDetected;
        if (config != undefined) {
            if (config.url != undefined) {
                this.url = config.url;
            }
        }
        this._wallet = new wallet_1.FlarexWallet(this.url);
        fetch(`${this.url}/wallet/v1/health`).then(resp => {
            if (resp.status == 200) {
                this.readyState = wallet_adapter_base_1.WalletReadyState.Installed;
            }
            else {
                this.readyState = wallet_adapter_base_1.WalletReadyState.Unsupported;
            }
        }).catch(() => {
            this.readyState = wallet_adapter_base_1.WalletReadyState.Unsupported;
        });
    }
    autoConnect() {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.readyState != wallet_adapter_base_1.WalletReadyState.Installed) {
                return;
            }
            try {
                const pubkey = localStorage.getItem('flarex_wallet_pubkey');
                if (pubkey == null) {
                    throw new wallet_adapter_base_1.WalletNotConnectedError();
                }
                const publicKey = new web3_js_1.PublicKey(pubkey);
                this._publicKey = publicKey;
                this.emit('connect', publicKey);
            }
            catch (err) {
                this.emit('error', err);
                throw err;
            }
        });
    }
    connect() {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                if (this.connected || this.connecting)
                    return;
                if (this.readyState != wallet_adapter_base_1.WalletReadyState.Installed) {
                    throw new wallet_adapter_base_1.WalletNotReadyError();
                }
                this._connecting = true;
                const wallet = this._wallet;
                if (wallet == null) {
                    throw new wallet_adapter_base_1.WalletNotConnectedError();
                }
                try {
                    const pubkey = yield wallet.getPublicKey();
                    localStorage.setItem('flarex_wallet_pubkey', pubkey.toBase58());
                    this._publicKey = pubkey;
                    this.emit('connect', pubkey);
                }
                catch (err) {
                    throw new wallet_adapter_base_1.WalletAccountError(err);
                }
            }
            catch (err) {
                this.emit('error', err);
                throw err;
            }
            finally {
                this._connecting = false;
            }
        });
    }
    disconnect() {
        return __awaiter(this, void 0, void 0, function* () {
            localStorage.removeItem('flarex_wallet_pubkey');
            this._publicKey = null;
            this.emit('disconnect');
        });
    }
    signMessage(message) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                const wallet = this._wallet;
                if (wallet == null) {
                    throw new wallet_adapter_base_1.WalletNotConnectedError();
                }
                try {
                    const sig = yield wallet.signMessage(message);
                    return sig;
                }
                catch (err) {
                    throw new wallet_adapter_base_1.WalletSignMessageError(err);
                }
            }
            catch (err) {
                this.emit('error', err);
                throw err;
            }
        });
    }
    signTransaction(transaction) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                const wallet = this._wallet;
                if (wallet == null) {
                    throw new wallet_adapter_base_1.WalletNotConnectedError();
                }
                try {
                    const tx = yield wallet.signTransaction(transaction);
                    return tx;
                }
                catch (err) {
                    throw new wallet_adapter_base_1.WalletSignTransactionError(err);
                }
            }
            catch (err) {
                this.emit('error', err);
                throw err;
            }
        });
    }
    get wallet() {
        return this._wallet;
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
    set readyState(readyState) {
        this._readyState = readyState;
        this.emit('readyStateChange', readyState);
    }
}
exports.FlarexWalletAdapter = FlarexWalletAdapter;
