import { Injectable } from '@angular/core';
import { 
  BehaviorSubject, Observable, 
  asyncScheduler, catchError, from, forkJoin, map, of, scheduled, mergeAll, reduce, 
} from 'rxjs';

import { 
  AccountInfo, Connection, PublicKey, Version, 
  Transaction as LegacyTransaction, TransactionInstruction, TransactionMessage, VersionedTransaction, 
  TransactionConfirmationStrategy, TransactionSignature, SignatureResult, RpcResponseAndContext, 
  clusterApiUrl, 
  LAMPORTS_PER_SOL, 
} from '@solana/web3.js';
import { 
  AccountLayout, Mint, RawAccount, 
  getMint, getTokenMetadata, 
  TOKEN_PROGRAM_ID, TOKEN_2022_PROGRAM_ID, 
} from '@solana/spl-token';
import { TokenMetadata } from '@solana/spl-token-metadata';
import { SendTransactionOptions, WalletAdapterNetwork } from '@solana/wallet-adapter-base';
import { getFavoriteDomain } from '@bonfida/spl-name-service';

@Injectable({
  providedIn: 'root'
})
export class SolanaService {
  private _network = WalletAdapterNetwork.Devnet;
  private _connection: Connection;
  private _connectionChangeSubject = new BehaviorSubject<Connection | null>(null);

  connectionChange = this._connectionChangeSubject.asObservable();

  constructor() {
    let endpoint = clusterApiUrl(this._network);

    this._connection = new Connection(endpoint, 'confirmed');
  }

  getVersion(): Observable<Version> {
    return from(this.connection.getVersion())
  }

  getAccount(pubkey: PublicKey | null): Observable<string> {
    if (pubkey == null) {
      return of('');
    }

    return from(
      getFavoriteDomain(this.connection, pubkey),
    ).pipe(
      map(({ reverse }) => reverse + ".sol"),
      catchError(() => of(pubkey.toBase58())),
    );
  }

  getBalance(pubkey: PublicKey | null): Observable<number> {
    if (pubkey == null) {
      return of(0);
    }

    return from(
      this.connection.getBalance(pubkey)
    ).pipe(
      map((balance) => balance / LAMPORTS_PER_SOL)
    );
  }

  requestAirdrop(to: PublicKey, amount: number): Observable<string> {
    const lamports = amount * LAMPORTS_PER_SOL;

    return from(this.connection.requestAirdrop(to, lamports));
  }

  getToken(mint: PublicKey, programId: PublicKey = TOKEN_PROGRAM_ID): Observable<Mint> {
    return from(getMint(this.connection, mint, undefined, programId));
  }

  getTokenMetadata(mint: PublicKey): Observable<TokenMetadata | null> {
    return from(getTokenMetadata(this.connection, mint));
  }

  getTokenAccountsByOwner(owner: PublicKey | null): Observable<AssociatedTokenAccount[]> {
    if (owner == null) {
      return of([]);
    }

    return scheduled([
      from(this.connection.getTokenAccountsByOwner(owner, { programId: TOKEN_PROGRAM_ID })),
      from(this.connection.getTokenAccountsByOwner(owner, { programId: TOKEN_2022_PROGRAM_ID })),
    ], asyncScheduler).pipe(
      mergeAll(),
      map((accounts) => accounts.value.flatMap(
        (raw) => {
          const account = {
            pubkey: raw.pubkey,
            info: raw.account,
            data: AccountLayout.decode(raw.account.data),
          };

          const ata = ToATA(account);

          let observable: Observable<Token>;
          if (!ata.isToken2022()) {
            observable = this.getToken(ata.mint).pipe(
              map((mint) => ({ mint, metadata: null }))
            );
          } else {
            observable = forkJoin({
              mint: this.getToken(ata.mint, TOKEN_2022_PROGRAM_ID),
              metadata: this.getTokenMetadata(ata.mint),
            }).pipe(
              map(({ mint, metadata }) => ({ mint, metadata }))
            );
          }

          observable.subscribe(
            (token) => ata.token = token 
          );
          
          return ata;
        }
      )),
      reduce((acc, value) => acc.concat(value)),
    );
  }

  transfer(source: PublicKey, destination: PublicKey, amount: number, owner: PublicKey, instructions: TransactionInstruction[]): Observable<Transaction> {
    return from(
      this.connection.getLatestBlockhashAndContext(),
    ).pipe(
      map((result) => {
        const keys = [
          { pubkey: source, isSigner: false, isWritable: true },
          { pubkey: destination, isSigner: false, isWritable: true },
          { pubkey: owner, isSigner: true, isWritable: false },
        ];

        // 3 => Self::Transfer { amount }
        const data = Buffer.alloc(9);
        data.writeUInt8(3);                      // cmd (1 byte)
        data.writeBigInt64LE(BigInt(amount), 1); // amount (8 bytes)

        instructions.push(new TransactionInstruction(
          { keys, programId: TOKEN_PROGRAM_ID , data }
        )) ;

        const message = new TransactionMessage({
          payerKey: owner,
          recentBlockhash: result.value.blockhash,
          instructions: instructions,
        }).compileToV0Message();

        const vtx = new VersionedTransaction(message);

        return { transaction: vtx, options: { minContextSlot: result.context.slot } };
      })
    )
  }

  sendTransaction(tx: Transaction): Observable<string> {
    let observable: Observable<string>;
    if (tx.transaction instanceof VersionedTransaction) {
      observable = from(this.connection.sendTransaction(tx.transaction, tx.options));
    } else {
      observable = from(this.connection.sendTransaction(tx.transaction, [], tx.options));
    }

    return observable;
  }

  confirmTransaction(transaction: TransactionConfirmationStrategy | TransactionSignature): Observable<{}> {
    let observable: Observable<RpcResponseAndContext<SignatureResult>>;

    if (typeof transaction == 'string') {
      const signature: TransactionSignature = transaction;
      observable = from(this.connection.confirmTransaction(signature));
    } else {
      const strategy = transaction;
      observable = from(this.connection.confirmTransaction(strategy));
    }

    return observable.pipe(
      map((result) => {
        const err = result.value.err;
        if ((err != null) && (typeof err == 'string')) {
          throw new Error(err)
        }

        return {};
      })
    );
  }

  public get network(): WalletAdapterNetwork {
    return this._network;
  }
  public set network(network: WalletAdapterNetwork) {
    this._network = network;
  }

  public get connection(): Connection {
    return this._connection;
  }
  public set connection(connection: Connection) {
    this._connection = connection;
    this._connectionChangeSubject.next(connection);
  }
}

export interface Transaction {
  transaction: VersionedTransaction | LegacyTransaction;
  options: SendTransactionOptions;
}

export interface Account<T> {
  pubkey: PublicKey;
  info: AccountInfo<Buffer>;
  data: T;
}

export function ToATA<T extends RawAccount>(
  account: Account<T>
): AssociatedTokenAccount {
  return new AssociatedTokenAccount(account);
}

export class AssociatedTokenAccount {
  private _account: Account<RawAccount>;
  private _token: Token | undefined;

  constructor(account: Account<RawAccount>) {
    this._account = account;
  }

  public isToken2022(): boolean {
    return this._account.info.owner.equals(TOKEN_2022_PROGRAM_ID);
  }

  public get account(): RawAccount {
    return this._account.data;
  }

  public get pubkey(): PublicKey {
    return this._account.pubkey;
  }

  public get mint(): PublicKey {
    return this.account.mint;
  }

  public get display_mint(): string {
    const token = this.token;
    if ((token != null) && (token.metadata != null)) {
      return token.metadata.symbol;
    }

    return `${this.mint.toBase58().slice(0, 10)}...`;
  }

  public get amount(): number {
    const amount = Number(this.account.amount);
    return amount / Math.pow(10, this.decimals);
  }

  public set token(value: Token | undefined) {
    this._token = value;
  }

  public get token(): Token | undefined{
    return this._token;
  }

  public get decimals(): number {
    const token = this.token;
    if (token == undefined) {
      return 9;
    }

    return token.mint.decimals;
  }
}

export interface Token {
  mint: Mint;
  metadata: TokenMetadata | null;
}
