import { AsyncPipe, CurrencyPipe, DatePipe, DecimalPipe, TitleCasePipe } from '@angular/common';
import { Component, ChangeDetectorRef, HostListener, OnInit, ViewChild } from '@angular/core';
import { RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { Observable, catchError, concatMap, filter, forkJoin, interval, map, of, switchMap, take, combineLatest, startWith, timer } from 'rxjs';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatDividerModule } from '@angular/material/divider';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatMenuModule } from '@angular/material/menu';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatSnackBarModule, MatSnackBar } from '@angular/material/snack-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';

import { WalletMessage, WalletMessageType } from '@flarex/wallet-adapter';
import { WalletAdapterNetwork } from '@solana/wallet-adapter-base';

import { environment as env } from '../environments/environment';
import { IdentityService, User, SigninResult } from './identity.service';
import { SolanaService, AssociatedTokenAccount } from './solana.service';
import { WalletService } from './wallet.service';
import { PythService, SolanaPrice } from './pyth.service';
import { TokenTransferComponent } from './token-transfer/token-transfer.component';

declare var google: any;

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    AsyncPipe,
    CurrencyPipe,
    DatePipe,
    DecimalPipe,
    TitleCasePipe,
    RouterOutlet,
    MatButtonModule,
    MatButtonToggleModule,
    MatCardModule,
    MatDividerModule,
    MatIconModule,
    MatInputModule,
    MatMenuModule,
    MatSnackBarModule,
    MatSidenavModule,
    MatToolbarModule,
    MatTooltipModule,
  ],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent implements OnInit {
  user: User | undefined = undefined;
  lastSigninMethod: string | null = null;

  // 分開處理不同的資料流
  account: Observable<{ account: string, balance: number }> = of({ account: '', balance: 0 });
  solanaPrice: Observable<SolanaPrice | null> = of(null);
  tokenAccounts: Observable<AssociatedTokenAccount[]> = of([]);

  @ViewChild(RouterOutlet)
  outlet: RouterOutlet | undefined;

  currentUrl: string = '';

  constructor(
    private changeDetectorRef: ChangeDetectorRef,
    private router: Router,
    private snackBar: MatSnackBar,
    private identityService: IdentityService,
    private solanaService: SolanaService,
    private walletService: WalletService,
    private pythService: PythService,
  ) {
    // 1. 錢包基本資料 - 立即更新
    this.account = this.walletService.walletChange.pipe(
      switchMap((pubkey) => {
        console.log('Wallet changed:', pubkey?.toBase58());
        
        if (!pubkey) {
          console.log('No wallet public key available');
          return of({ account: '', balance: 0 });
        }

        console.log('Fetching account data for:', pubkey.toBase58());
        return forkJoin({
          account: this.solanaService.getAccount(pubkey),
          balance: this.solanaService.getBalance(pubkey),
        }).pipe(
          map(({ account, balance }) => {
            console.log('Account data fetched:', { account, balance });
            return { 
              account: account || pubkey.toBase58(), 
              balance 
            };
          }),
          catchError(error => {
            console.error('Error fetching account data:', error);
            return of({ account: pubkey.toBase58(), balance: 0 });
          })
        );
      }),
      startWith({ account: '', balance: 0 }) // 添加初始值
    );

    // 2. SOL 價格 - 獨立更新，每30秒刷新
    this.solanaPrice = timer(0, 30000).pipe(
      switchMap(() => this.pythService.getSolanaPrice()),
      catchError(error => {
        console.error('Error fetching price data:', error);
        return of(null);
      })
    );

    // 3. Token accounts - 獨立處理
    this.tokenAccounts = this.walletService.walletChange.pipe(
      switchMap((pubkey) => {
        if (!pubkey) {
          return of([]);
        }
        return this.solanaService.getTokenAccountsByOwner(pubkey).pipe(
          catchError(error => {
            console.error('Error fetching token accounts:', error);
            return of([]);
          })
        );
      })
    );

    // 路由變更監聽
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
    ).subscribe({
      next: (event) => this.currentUrl = event.url,
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    });

    this.lastSigninMethod = localStorage.getItem('last-signin-method');

    // 使用者身份驗證
    const token = localStorage.getItem('token');
    if (token == null) return;

    this.identityService.getUserFromToken(token).pipe(
      catchError((e) => {
        console.error('Token verification failed:', e);

        let passkeysToken = localStorage.getItem('passkeys-token');
        if (passkeysToken == null) {
          throw new Error('passkeys token not found');
        }

        return this.identityService.verifyPasskeyToken(passkeysToken).pipe(
          concatMap(() => this.signin('passkeys', passkeysToken)),
          catchError((e) => {
            console.error('Passkey verification failed:', e);
            localStorage.removeItem('passkeys-token');
            return this.passkeyLogin();
          }),
        );
      }),
    ).subscribe({
      next: (result) => {
        console.log('User signed in:', result.user);
        this.user = result.user;
        // 用戶登入成功後，觸發錢包資料載入
        this.initializeWallet();
      },
      error: (err) => console.error('Authentication error:', err),
      complete: () => console.log('Authentication complete'),
    });
  }

  ngOnInit() {
    console.log('AppComponent initialized');
    // 如果用戶已經存在，初始化錢包
    if (this.user) {
      this.initializeWallet();
    }
    
    const url = window.location.pathname;
    if (url.length > 1) {
      const params = decodeURIComponent(url.slice(1));
      this.handleProtocolHandler(params);
    }
  }

  // 新增錢包初始化方法
  private initializeWallet() {
    console.log('Initializing wallet for user:', this.user?.username);
    
    // 強制刷新錢包服務
    setTimeout(() => {
      console.log('Refreshing wallet...');
      this.walletService.refreshWallet();
    }, 100);
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
        complete: () => window.close(),
      });
    }
  }

  @HostListener('window:message', ['$event'])
  messageHandler(event: MessageEvent) {
    console.log(event);

    // Check if the wallet is ready
    if (event.data == 'IS_READY') {
      if (this.user != undefined) {
        event.source?.postMessage('WALLET_READY', { targetOrigin: event.origin });
      }
      return;
    }

    this.walletService.responseCallback = (resp) => {
      if (event.source) {
        event.source.postMessage(resp, { targetOrigin: event.origin });
      }
    }

    // Handle wallet messages
    const msg = event.data as WalletMessage;
    switch (msg.type) {
      case WalletMessageType.SIGN_MESSAGE:
        this.router.navigate(['/sign-message'], { 
          state: { msg }
        });
        break;

      default:
        this.walletService.messageHandler(msg)?.subscribe({
          next: (resp) => this.walletService.sendResponse(resp),
          error: (err) => console.error(err),
          complete: () => window.close(),
        });
        break;
    }
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

    this.signin('google', token).subscribe({
      next: (result) => {
        console.log('Google signin successful:', result.user);
        this.user = result.user;
        this.initializeWallet();
      },
      error: (err) => console.error('Google signin error:', err),
      complete: () => this.changeDetectorRef.detectChanges(),
    })
  }

  passkeyLogin(): Observable<SigninResult> {
    return this.identityService.directPasskeyLogin().pipe(
      concatMap((token) => this.identityService.verifyPasskeyToken(token).pipe(
        map(() => token)
      )),
      concatMap((token) => {
        console.log("Passkeys token: " + token);
        localStorage.setItem('passkeys-token', token);

        return this.signin('passkeys', token);
      })
    );
  }

  passkeyLoginHandler() {
    this.passkeyLogin().subscribe({
      next: (result) => {
        console.log('Passkey signin successful:', result.user);
        this.user = result.user;
        this.initializeWallet();
      },
      error: (err) => console.error('Passkey signin error:', err),
      complete: () => console.log('Passkey signin complete'),
    })
  }

  signin(provider: string, token: string): Observable<SigninResult> {
    return this.identityService.signin(provider, token).pipe(
      map((result) => {
        localStorage.setItem('last-signin-method', provider);
        localStorage.setItem('token', result.token.token);
        return result;
      })
    );
  }

  registerPasskey() {
    this.identityService.registerPasskey().subscribe({
      next: (user) => console.log(user),
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }

  logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('last-signin-method');
    localStorage.removeItem('passkeys-token');
    this.user = undefined;
    this.identityService.logout();
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

    this.snackBar.open('Account copied', undefined, {
      duration: 1000
    });
  }

  browseAccount(account: string, network: WalletAdapterNetwork) {
    window.open(`https://explorer.solana.com/address/${account}?cluster=${network}`, '_blank');
  }

  switchRouter(url: string) {
    this.router.navigateByUrl(url);
  }

  refreshTokens() {
    const outlet = this.outlet;
    if (outlet == undefined) return;

    if (outlet.component instanceof TokenTransferComponent) {
      outlet.component.refreshTokens();
    }
  }

  public get network(): WalletAdapterNetwork {
    return this.solanaService.network
  }

  // 新增格式化價格的方法
  formatPrice(price: number): string {
    return this.pythService.formatPrice(price);
  }

  // 優化的 tooltip 內容，利用更寬的空間
  getPriceTooltip(priceInfo: SolanaPrice): string {
    const price = this.formatPrice(priceInfo.price);
    const confidence = this.formatPrice(priceInfo.confidence);
    const lastUpdated = priceInfo.lastUpdated.toLocaleString('en-US', {
      month: 'short',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true
    });
    
    return `💰 SOL Price: ${price}
📊 Price Confidence: ±${confidence}
🕒 Last Updated: ${lastUpdated}
📡 Data Source: Pyth Network`;
  }

  // 新增計算 USD 價值的方法
  calculateUsdValue(balance: number, price: number | null): number {
    if (!price || !balance) return 0;
    return this.pythService.calculateUSDValue(balance, price);
  }
}
