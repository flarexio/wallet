import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable, from, map, of } from 'rxjs';

import { 
  Connection, PublicKey, Version, 
  TransactionInstruction, TransactionMessage, VersionedTransaction, 
  clusterApiUrl, 
  LAMPORTS_PER_SOL, 
} from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { SendTransactionOptions, WalletAdapterNetwork } from '@solana/wallet-adapter-base';

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

  requestAirdrop(to: PublicKey, lamports: number): Observable<string> {
    return from(this.connection.requestAirdrop(to, lamports));
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

        const tx = new VersionedTransaction(message);

        return new Transaction(tx, { minContextSlot: result.context.slot });
      }),
    )
  }

  sendTransaction(tx: Transaction): Observable<string> {
    return from(this.connection.sendTransaction(tx.transaction, tx.options));
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

export class Transaction {
  transaction: VersionedTransaction;
  options: SendTransactionOptions;

  constructor(tx: VersionedTransaction, opts: SendTransactionOptions) {
    this.transaction = tx;
    this.options = opts;
  }
}
