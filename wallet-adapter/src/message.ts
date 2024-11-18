export enum WalletMessageType {
  TRUST_SITE = 'TRUST_SITE',
  SIGN_TRANSACTION = 'SIGN_TRANSACTION',
  SIGN_MESSAGE = 'SIGN_MESSAGE',
}

export class WalletMessage {
  id: string;
  type: WalletMessageType;
  origin: string;
  payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;

  constructor(
    id: string,
    type: WalletMessageType,
    origin: string,
    payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload
  ) {
    this.id = id;
    this.type = type;
    this.origin = origin;
    this.payload = payload;
  }

  serialize(): string {
    let payload: string;
    switch (this.type) {
      case WalletMessageType.TRUST_SITE:
        const trustSitePayload = this.payload as TrustSitePayload;
        payload = trustSitePayload.serialize();
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        const signTransactionPayload = this.payload as SignTransactionPayload;
        payload = signTransactionPayload.serialize();
        break;

      case WalletMessageType.SIGN_MESSAGE:
        const signMessagePayload = this.payload as SignMessagePayload;
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

  static deserialize(message: string): WalletMessage {
    const value = JSON.parse(message);

    let payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
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

    return new WalletMessage(
      value.id,
      value.type,
      value.origin,
      payload,
    );
  }
}

export class TrustSitePayload {
  app: string;
  domain: string;
  icon?: string;
  accept?: boolean;
  pubkey?: Uint8Array;

  constructor(app: string, domain: string, icon?: string, accept?: boolean, pubkey?: Uint8Array) {
    this.app = app;
    this.domain = domain;
    this.icon = icon;
    this.accept = accept;
    this.pubkey = pubkey;
  }

  serialize(): string {
    let pubkey: string | undefined = undefined;
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

  static deserialize(payload: string): TrustSitePayload {
    const value = JSON.parse(payload);

    let pubkey: Uint8Array | undefined = undefined;
    if (value.pubkey != undefined) {
      pubkey = Buffer.from(value.pubkey, 'base64');
    }

    return new TrustSitePayload(
      value.app,
      value.domain,
      value.icon,
      value.accept,
      pubkey,
    );
  }
}

export class SignTransactionPayload {
  tx: Uint8Array;
  sigs?: Uint8Array[];

  constructor(tx: Uint8Array, sigs?: Uint8Array[]) {
    this.tx = tx;
    this.sigs = sigs;
  }

  serialize(): string {
    const tx = Buffer.from(this.tx).toString('base64');

    let sigs: string[] | undefined = undefined;
    if (this.sigs != undefined) {
      sigs = this.sigs.map((sig) => Buffer.from(sig).toString('base64'));
    }

    return JSON.stringify({ tx, sigs });
  }

  static deserialize(payload: string): SignTransactionPayload {
    const value = JSON.parse(payload);

    const tx = Buffer.from(value.tx, 'base64');

    let sigs: Uint8Array[] | undefined = undefined;
    if (value.sigs != undefined) {
      sigs = value.sigs.map((sig: string) => Buffer.from(sig, 'base64'));
    }

    return new SignTransactionPayload(tx, sigs);
  }
}

export class SignMessagePayload {
  msg: Uint8Array;
  sig?: Uint8Array;

  constructor(msg: Uint8Array, sig?: Uint8Array) {
    this.msg = msg;
    this.sig = sig;
  }

  serialize(): string {
    const msg = Buffer.from(this.msg).toString('base64');

    let sig: string | undefined = undefined;
    if (this.sig != undefined) {
      sig = Buffer.from(this.sig).toString('base64');
    }

    return JSON.stringify({ msg, sig });
  }

  static deserialize(payload: string): SignMessagePayload {
    const value = JSON.parse(payload);

    const msg = Buffer.from(value.msg, 'base64');

    let sig: Uint8Array | undefined = undefined;
    if (value.sig != undefined) {
      sig = Buffer.from(value.sig, 'base64');
    }

    return new SignMessagePayload(msg, sig);
  }
}

export class WalletMessageResponse {
  id: string;
  type: WalletMessageType;
  success: boolean;
  error?: string;
  payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload;

  constructor(
    id: string,
    type: WalletMessageType,
    success: boolean,
    error?: string,
    payload?: TrustSitePayload | SignTransactionPayload | SignMessagePayload
  ) {
    this.id = id;
    this.type = type;
    this.success = success;
    this.error = error;
    this.payload = payload;
  }

  serialize(): string {
    let payload: string;
    switch (this.type) {
      case WalletMessageType.TRUST_SITE:
        payload = (this.payload as TrustSitePayload).serialize();
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        payload = (this.payload as SignTransactionPayload).serialize();
        break;

      case WalletMessageType.SIGN_MESSAGE:
        payload = (this.payload as SignMessagePayload).serialize();
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

  static deserialize(message: string): WalletMessageResponse {
    const value = JSON.parse(message);

    let payload: TrustSitePayload | SignTransactionPayload | SignMessagePayload;
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

    return new WalletMessageResponse(
      value.id,
      value.type,
      value.success,
      value.error,
      payload,
    );
  }
}
