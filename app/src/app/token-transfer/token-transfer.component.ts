import { AsyncPipe } from '@angular/common';
import { Component } from '@angular/core';
import { 
  FormBuilder, FormGroup, FormsModule, ReactiveFormsModule, 
  AbstractControl, ValidationErrors, ValidatorFn, Validators, 
} from '@angular/forms';
import { Observable, concatMap, map, of } from 'rxjs';

import { MatButtonModule } from '@angular/material/button';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBarModule, MatSnackBar } from '@angular/material/snack-bar';

import { PublicKey, TransactionInstruction } from '@solana/web3.js';
import { 
  createAssociatedTokenAccountInstruction, 
  getAccount, getAssociatedTokenAddressSync, 
} from '@solana/spl-token';
import { WalletAdapterNetwork } from '@solana/wallet-adapter-base';
import { v4 as uuid } from 'uuid';

import { IdentityService } from '../identity.service';
import { SolanaService, AssociatedTokenAccount } from '../solana.service';
import { WalletService } from '../wallet.service';
import { TransactionSnackbarComponent } from '../transaction-snackbar/transaction-snackbar.component';
import { NumberFormatPipe } from '../shared/number-format.pipe';

@Component({
  selector: 'app-token-transfer',
  standalone: true,
  imports: [
    AsyncPipe,
    NumberFormatPipe,
    FormsModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatExpansionModule,
    MatIconModule,
    MatInputModule,
    MatSnackBarModule,
  ],
  templateUrl: './token-transfer.component.html',
  styleUrl: './token-transfer.component.scss'
})
export class TokenTransferComponent {
  forms: FormGroup[] = [];

  tokenAccounts: Observable<AssociatedTokenAccount[]> = of([]);

  constructor(
    private formBuilder: FormBuilder,
    private snackBar: MatSnackBar,
    private identityService: IdentityService,
    private solanaService: SolanaService,
    private walletService: WalletService,
  ) {
    this.tokenAccounts = this.walletService.walletChange.pipe(
      concatMap((pubkey) => this.solanaService.getTokenAccountsByOwner(pubkey).pipe(
        map((accounts) => {
          this.forms = accounts.map((account) => this.formBuilder.group({
            toWallet: ['', [
              Validators.required,
              this.walletValidator,
            ]],
            amount: ['', [
              Validators.required,
              this.amountValidator(account),
            ]]
          }));

          return accounts;
        })
      )),
    );
  }

  walletValidator(control: AbstractControl): ValidationErrors | null {
    const address: string = control.value;
    if (!address) {
      return null;
    }

    try {
      new PublicKey(address)
    } catch (e) {
      return { 'wallet': e };
    }

    return null;
  }

  amountValidator(account: AssociatedTokenAccount): ValidatorFn {
    return (control) => {
      const amountStr: string = control.value;
      if (!amountStr) {
        return null;
      }

      const amount = Number(amountStr);
      if (isNaN(amount)) {
        return { 'amount': 'NaN' };
      }

      if (amount > account.amount) {
        return { 'amount': 'Insufficient balance' };
      }

      return null;
    }
  }

  getWalletError(wallet: AbstractControl | null) {
    if (wallet == null) {
      return '';
    }

    if (wallet.hasError('required')) {
      return 'You must enter the receiving wallet';
    }

    if (wallet.hasError('wallet')) {
      return 'Invalid wallet address';
    }

    return '';
  }

  getAmountError(amount: AbstractControl | null) {
    if (amount == null) {
      return '';
    }

    if (amount.hasError('required')) {
      return 'You must enter the transfer amount';
    }

    if (amount.hasError('amount')) {
      return amount.getError('amount');
    }

    return '';
  }

  transfer(from: AssociatedTokenAccount, target: FormGroup) {
    const owner = this.walletService.currentWallet;
    if (owner == null) return;

    const mint = from.mint;
    console.log(`mint: ${mint}`);

    const fromWallet = owner;
    const fromATA = getAssociatedTokenAddressSync(mint, owner);
    console.log(`fromWallet: ${fromWallet}`);
    console.log(`fromATA: ${fromATA}`);

    const toWalletControl = target.get('toWallet');
    if (toWalletControl == null) return;

    const toWallet = new PublicKey(toWalletControl.value);
    const toATA = getAssociatedTokenAddressSync(mint, toWallet);
    console.log(`toWallet: ${toWallet}`);
    console.log(`toATA: ${toATA}`);

    const transactionInstructions: TransactionInstruction[] = [];
    getAccount(this.solanaService.connection, toATA).catch(
      (e) => transactionInstructions.push(
        createAssociatedTokenAccountInstruction(
          owner,
          toATA,
          toWallet,
          mint,
        )
      )
    )

    const source = fromATA;
    const destination = toATA;

    const amountControl = target.get('amount');
    if (amountControl == null) return;

    const amount = amountControl.value * Math.pow(10, from.decimals);

    const tid = uuid();

    this.solanaService.transfer(
      source, 
      destination, 
      amount, 
      owner,
      transactionInstructions, 
    ).pipe(
      concatMap((tx) => this.walletService.signTransaction(tid, tx.transaction).pipe(
        map((resp) => { tx.transaction = resp.tx; return tx; }),
      )),
      concatMap((tx) => this.solanaService.sendTransaction(tx)),
    ).subscribe({
      next: (signature) => {
        this.snackBar.openFromComponent(TransactionSnackbarComponent, {
          data: {
            signature,
            network: this.network,
            action: () => this.identityService.refreshUser(),
          }
        })
      },
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    });
  }

  public get network(): WalletAdapterNetwork {
    return this.solanaService.network
  }
}
