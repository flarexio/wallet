import { SlicePipe } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';

import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBarModule, MatSnackBarRef, MAT_SNACK_BAR_DATA } from '@angular/material/snack-bar';

import { WalletAdapterNetwork } from '@solana/wallet-adapter-base';

import { SolanaService } from '../solana.service';

@Component({
  selector: 'transaction-snackbar',
  standalone: true,
  imports: [
    SlicePipe,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
  ],
  templateUrl: './transaction-snackbar.component.html',
  styleUrl: './transaction-snackbar.component.scss'
})
export class TransactionSnackbarComponent implements OnInit {
  readonly snackBarRef = inject(MatSnackBarRef<TransactionSnackbarComponent>);
  readonly data = inject<SnackbarData>(MAT_SNACK_BAR_DATA);

  isConfirmed = false;
  err: Error | undefined;

  constructor(
    private solanaService: SolanaService,
  ) {}

  ngOnInit(): void {
    this.solanaService.confirmTransaction(this.data.signature).subscribe({
      next: () => this.isConfirmed = true,
      error: (err) => this.err = err,
      complete: () => this.data.action(),
    });
  }
}

interface SnackbarData {
  signature: string;
  network: WalletAdapterNetwork;
  action: () => void;
}
