export enum WalletMessageType {
  TRUST_SITE = 'TRUST_SITE',
  SIGN_TRANSACTION = 'SIGN_TRANSACTION',
  SIGN_MESSAGE = 'SIGN_MESSAGE',
}

export type WalletMessagePayload = TrustSitePayload | SignMessagePayload | SignTransactionPayload;

export class WalletMessage {
  id: string;
  type: WalletMessageType;
  origin: string;
  payload: WalletMessagePayload;

  constructor(
    id: string,
    type: WalletMessageType,
    origin: string,
    payload: WalletMessagePayload,
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

      case WalletMessageType.SIGN_MESSAGE:
        const signMessagePayload = this.payload as SignMessagePayload;
        payload = signMessagePayload.serialize();
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        const signTransactionPayload = this.payload as SignTransactionPayload;
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

  static deserialize(message: string): WalletMessage {
    const value = JSON.parse(message);

    let payload: WalletMessagePayload;
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

export class SignMessagePayload {
  message: Uint8Array;
  signature?: Uint8Array;

  constructor(message: Uint8Array, signature?: Uint8Array) {
    this.message = message;
    this.signature = signature;
  }

  serialize(): string {
    const message = Buffer.from(this.message).toString('base64');

    let signature: string | undefined = undefined;
    if (this.signature != undefined) {
      signature = Buffer.from(this.signature).toString('base64');
    }

    return JSON.stringify({ message, signature });
  }

  static deserialize(payload: string): SignMessagePayload {
    const value = JSON.parse(payload);

    const message = Buffer.from(value.message, 'base64');

    let signature: Uint8Array | undefined = undefined;
    if (value.signature != undefined) {
      signature = Buffer.from(value.signature, 'base64');
    }

    return new SignMessagePayload(message, signature);
  }
}

export class SignTransactionPayload {
  transaction: Uint8Array;
  versioned: boolean;
  signatures?: Uint8Array[];

  constructor(transaction: Uint8Array, versioned: boolean, signatures?: Uint8Array[]) {
    this.transaction = transaction;
    this.versioned = versioned;
    this.signatures = signatures;
  }

  serialize(): string {
    const transaction = Buffer.from(this.transaction).toString('base64');
    const versioned = this.versioned;

    let signatures: string[] | undefined = undefined;
    if (this.signatures != undefined) {
      signatures = this.signatures.map(
        (sig) => Buffer.from(sig).toString('base64'),
      );
    }

    return JSON.stringify({ transaction, versioned, signatures });
  }

  static deserialize(payload: string): SignTransactionPayload {
    const value = JSON.parse(payload);

    const transaction = Buffer.from(value.transaction, 'base64');
    const versioned = value.versioned;

    let signatures: Uint8Array[] | undefined = undefined;
    if (value.signatures != undefined) {
      signatures = value.signatures.map(
        (sig: string) => Buffer.from(sig, 'base64'),
      );
    }

    return new SignTransactionPayload(transaction, versioned, signatures);
  }
}

export class WalletMessageResponse {
  id: string;
  type: WalletMessageType;
  success: boolean;
  payload?: WalletMessagePayload;
  error?: string;

  constructor(
    id: string,
    type: WalletMessageType,
    success: boolean,
    payload?: WalletMessagePayload,
    error?: string,
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

      case WalletMessageType.SIGN_MESSAGE:
        payload = (this.payload as SignMessagePayload).serialize();
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        payload = (this.payload as SignTransactionPayload).serialize();
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

      case WalletMessageType.SIGN_MESSAGE:
        payload = SignMessagePayload.deserialize(value.payload);
        break;

      case WalletMessageType.SIGN_TRANSACTION:
        payload = SignTransactionPayload.deserialize(value.payload);
        break;

      default:
        throw new Error(`unknown message type: ${value.type}`);
    }

    return new WalletMessageResponse(
      value.id,
      value.type,
      value.success,
      payload,
      value.error,
    );
  }
}
