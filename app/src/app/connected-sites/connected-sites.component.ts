import { Component } from '@angular/core';
import { DatePipe } from '@angular/common';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { PublicKey } from '@solana/web3.js';

import { TrustService, TrustedSite } from '../trust.service';
import { WalletService } from '../wallet.service';

@Component({
  selector: 'app-connected-sites',
  standalone: true,
  imports: [
    DatePipe,
    MatButtonModule,
    MatCardModule,
    MatIconModule,
    MatListModule,
    MatSnackBarModule,
  ],
  templateUrl: './connected-sites.component.html',
  styleUrl: './connected-sites.component.scss'
})
export class ConnectedSitesComponent {
  wallet: PublicKey | null = null;
  sites: TrustedSite[] = [];

  constructor(
    private snackBar: MatSnackBar,
    private trust: TrustService,
    private walletService: WalletService,
  ) {
    this.walletService.walletChange.subscribe({
      next: (wallet) => {
        this.wallet = wallet;
        this.refresh();
      },
      error: (err) => console.error(err),
    });
  }

  refresh() {
    this.sites = this.trust.list(this.wallet);
  }

  revoke(site: TrustedSite) {
    if (this.wallet == null) return;

    this.trust.revoke(this.wallet, site.origin);
    this.refresh();

    this.snackBar.open(`Disconnected ${site.origin}`, undefined, {
      duration: 2000,
    });
  }

  revokeAll() {
    if (this.wallet == null) return;

    this.trust.revokeAll(this.wallet);
    this.refresh();

    this.snackBar.open('Disconnected every site', undefined, {
      duration: 2000,
    });
  }
}
