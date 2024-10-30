import { JsonPipe } from '@angular/common';
import { Component, ChangeDetectorRef, HostListener } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Observable, catchError, concatMap, from } from 'rxjs';
import * as jose from 'jose';

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

declare var google: any;

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
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
  title = 'app';

  user: User | undefined = undefined;

  lastSigninMethod: string | null = null;

  constructor(
    private ref: ChangeDetectorRef,
    private identityService: IdentityService,
  ) {
    this.lastSigninMethod = localStorage.getItem('last-signin-method');

    let token = localStorage.getItem('passkeys-token');
    if (token == null) return;

    from(jose.jwtVerify(token, this.identityService.JWKS, 
      { audience: 'wallet.flarex.io' }
    )).pipe(
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
    })
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
    console.log("Google token: " + token);

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
}
