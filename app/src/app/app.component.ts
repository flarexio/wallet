import { AsyncPipe, CurrencyPipe, SlicePipe } from '@angular/common';
import { Component, ChangeDetectorRef, HostListener, OnInit } from '@angular/core';
import { Observable, catchError, concatMap, filter, interval, map, of, switchMap, take } from 'rxjs';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatMenuModule } from '@angular/material/menu';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatSnackBarModule, MatSnackBar } from '@angular/material/snack-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';

import { WalletMessage } from '@flarex/wallet-adapter';
import { WalletAdapterNetwork } from '@solana/wallet-adapter-base';

import { environment as env } from '../environments/environment';
import { IdentityService, User } from './identity.service';
import { SolanaService, AssociatedTokenAccount } from './solana.service';
import { WalletService } from './wallet.service';
import { TokenTransferComponent } from './token-transfer/token-transfer.component';

declare var google: any;

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    AsyncPipe,
    CurrencyPipe,
    SlicePipe,
    MatButtonModule,
    MatButtonToggleModule,
    MatCardModule,
    MatIconModule,
    MatInputModule,
    MatMenuModule,
    MatSnackBarModule,
    MatSidenavModule,
    MatToolbarModule,
    MatTooltipModule,
    TokenTransferComponent,
  ],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent implements OnInit {
  user: User | undefined = undefined;
  lastSigninMethod: string | null = null;

  account: Observable<string> = of('');
  balance: Observable<number> = of(0);
  tokenAccounts: Observable<AssociatedTokenAccount[]> = of([]);

  constructor(
    private changeDetectorRef: ChangeDetectorRef,
    private snackBar: MatSnackBar,
    private identityService: IdentityService,
    private solanaService: SolanaService,
    private walletService: WalletService,
  ) {
    this.account = this.walletService.walletChange.pipe(
      concatMap((pubkey) => this.solanaService.getAccount(pubkey)),
    );

    this.balance = this.walletService.walletChange.pipe(
      concatMap((pubkey) => this.solanaService.getBalance(pubkey)),
    );

    this.lastSigninMethod = localStorage.getItem('last-signin-method');

    let token = localStorage.getItem('passkeys-token');
    if (token == null) return;

    this.identityService.verifyToken(token).pipe(
      concatMap(() => this.identityService.signin('passkeys', token)),
      catchError((e) => {
        console.error(e);

        localStorage.removeItem('passkeys-token');
        return this.login();
      }),
    ).subscribe({
      next: (user) => {
        this.user = user;
        localStorage.setItem('last-signin-method', 'passkeys');
      },
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    });
  }

  ngOnInit() {
    const url = window.location.pathname;
    if (url.length > 1) {
      const params = decodeURIComponent(url.slice(1));
      this.handleProtocolHandler(params);
    }
  }

  private handleProtocolHandler(resource: string) {
    if (resource.startsWith('web+flarex:')) {
      const url = new URL(resource);
      if (!url.pathname.includes('wallet')) {
        return;
      }

      const session = url.searchParams.get('session');
      if (session == null) {
        return;
      }

      interval(1000).pipe(
        filter(() => this.user != undefined),
        take(1),
        switchMap(() => this.walletService.session(session)),
        concatMap((msg) => this.walletService.messageHandler(msg)),
        concatMap((resp) => this.walletService.ackSession(session, resp)),
      ).subscribe({
        next: (ok) => console.log(ok),
        error: (err) => console.error(err),
        complete: () => console.log('complete'),
      });
    }
  }

  @HostListener('window:message', ['$event'])
  messageHandler(event: MessageEvent) {
    console.log(event);

    // Check if the wallet is ready
    if (event.data == 'IS_READY') {
      if (this.user != null) {
        event.source?.postMessage('WALLET_READY', { targetOrigin: event.origin });
      }
      return;
    }

    // Handle wallet messages
    const msg = WalletMessage.deserialize(event.data);
    this.walletService.messageHandler(msg).subscribe({
      next: (resp) => {
        if (event.source) {
          event.source.postMessage(resp.serialize(), { targetOrigin: event.origin });
        }
      },
      error: (err) => console.error(err),
      complete: () => window.close(),
    });
  }


  @HostListener('window:load', ['$event'])
  loadHandler($event: Event) {
    google.accounts.id.initialize({
      client_id: env.GOOGLE_CLIENT_ID,
      callback: (response: any) => this.handleCredentialResponse(response)
    });

    google.accounts.id.renderButton(
      document.getElementById("googleSignIn"),
      { 
        type: "standard",
        theme: "outline", 
        size: "large",
        text: "signin_with",
        shape: "circle",
        logo_alignment: "left"
      }
    );

    if (this.lastSigninMethod != 'passkeys') {
      google.accounts.id.prompt();
    }
  }

  handleCredentialResponse(response: any) {
    const token: string = response.credential;

    this.identityService.signin('google', token).subscribe({
      next: (user) => {
        this.user = user;
        localStorage.setItem('last-signin-method', 'google');
      },
      error: (err) => console.error(err),
      complete: () => this.changeDetectorRef.detectChanges(),
    })
  }

  login(): Observable<User> {
    return this.identityService.directPasskeyLogin().pipe(
      concatMap((token) => this.identityService.verifyToken(token).pipe(
        map(() => token)
      )),
      concatMap((token) => {
        console.log("Passkeys token: " + token);
        localStorage.setItem('passkeys-token', token);

        return this.identityService.signin('passkeys', token);
      })
    );
  }

  loginHandler() {
    this.login().subscribe({
      next: (user) => {
        this.user = user;
        localStorage.setItem('last-signin-method', 'passkeys');
      },
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }

  registerPasskey() {
    this.identityService.registerPasskey().subscribe({
      next: (user) => console.log(user),
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }

  requestAirdrop() {
    const to = this.walletService.currentWallet;
    if (to == null) return;

    const amount = 2;

    this.solanaService.requestAirdrop(to, amount).subscribe({
      next: (tx) => console.log(tx),
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }

  async copyAccount(account: string) {
    await navigator.clipboard.writeText(account);

    this.snackBar.open('account copied', undefined, {
      duration: 1000
    });
  }

  browseAccount(account: string, network: WalletAdapterNetwork) {
    window.open(`https://explorer.solana.com/address/${account}?cluster=${network}`, '_blank');
  }

  public get network(): WalletAdapterNetwork {
    return this.solanaService.network
  }
}
