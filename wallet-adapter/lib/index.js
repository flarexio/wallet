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
exports.WalletMessageType = exports.FlareXWalletAdapter = exports.FlareXWalletName = void 0;
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
