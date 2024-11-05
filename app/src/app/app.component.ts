import { AsyncPipe, CurrencyPipe, JsonPipe } from '@angular/common';
import { Component, ChangeDetectorRef, HostListener } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Observable, catchError, concatMap, map, of } from 'rxjs';

import { PublicKey, LAMPORTS_PER_SOL } from '@solana/web3.js';
import { getAssociatedTokenAddressSync } from '@solana/spl-token';
import { v4 as uuid } from 'uuid';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';

import { environment as env } from '../environments/environment';
import { IdentityService, User } from './identity.service';
import { SolanaService } from './solana.service';
import { WalletService } from './wallet.service';

declare var google: any;

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    AsyncPipe,
    CurrencyPipe,
    JsonPipe,
    RouterOutlet,
    MatButtonModule,
    MatButtonToggleModule,
    MatCardModule,
    MatIconModule,
    MatMenuModule,
    MatSidenavModule,
    MatToolbarModule,
    MatTooltipModule,
  ],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  user: User | undefined = undefined;
  lastSigninMethod: string | null = null;

  wallet: Observable<PublicKey | null> = of(null);
  balance: Observable<number> = of(0);
  sig: string = '';

  constructor(
    private ref: ChangeDetectorRef,
    private identityService: IdentityService,
    private solanaService: SolanaService,
    private walletService: WalletService,
  ) {
    this.wallet = this.walletService.walletChange;

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
      complete: () => this.ref.detectChanges(),
    })
  }

  @HostListener('window:message', ['$event'])
  messageHandler($event: MessageEvent) {
    console.log($event);
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

    const lamports = 2 * LAMPORTS_PER_SOL;

    this.solanaService.requestAirdrop(to, lamports).subscribe({
      next: (tx) => console.log(tx),
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }

  transfer() {
    const owner = this.walletService.currentWallet;
    if (owner == null) return;

    const mint = new PublicKey('CcTqnRJaUXoZEHPkMz85gojwh1MSriwkJTfrKiJuwWGY');
    const fromATA = getAssociatedTokenAddressSync(mint, owner);
    console.log(`fromATA: ${fromATA}`);

    const toWallet = new PublicKey('BdTg8ZHfrUsYQWKygqrBjf9mUo2sKNkkm5jHbKesortd');
    const toATA = getAssociatedTokenAddressSync(mint, toWallet);
    console.log(`toATA: ${toATA}`);

    const source = fromATA;
    const destination = toATA;
    const amount = 1000000 * Math.pow(10, 9);

    const tid = uuid();

    this.solanaService.transfer(
      source, 
      destination, 
      amount, 
      owner,
      [], 
    ).pipe(
      concatMap((tx) => this.walletService.signTransaction(tid, tx)),
      concatMap(({ token, tx }) => this.solanaService.sendTransaction(tx)),
    ).subscribe({
      next: (sig) => this.sig = sig,
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    });
  }
}
