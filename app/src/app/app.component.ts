import { Component, HostListener } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Observable, catchError, concatMap, from, map } from 'rxjs';
import * as jose from 'jose';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';

import { environment as env } from '../environments/environment';
import { IdentityService } from './identity.service';
import { PasskeysService } from './passkeys.service';

declare var google: any;

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    RouterOutlet,
    MatButtonModule,
    MatCardModule,
    MatIconModule,
    MatSidenavModule,
    MatToolbarModule,
  ],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  title = 'app';
  name: string | undefined = undefined;
  message = '';

  constructor(
    private identityService: IdentityService,
    private passkeysService: PasskeysService,
  ) {
    let token = localStorage.getItem('token');
    if (token == null) return;

    from(jose.jwtVerify(token, this.passkeysService.JWKS, 
      { audience: 'wallet.flarex.io' }
    )).pipe(
      map(({ payload }) => payload),
      catchError((e) => {
        console.error(e);

        localStorage.removeItem('token');
        return this.login();
      }),
    ).subscribe({
      next: (payload) => this.name = payload.sub,
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
        text: "continue_with",
        shape: "rectangular",
        logo_alignment: "left"
      }
    );

    google.accounts.id.prompt();
  }

  handleCredentialResponse(response: any) {
    const token: string = response.credential;
    console.log("Encoded JWT ID token: " + token);

    this.identityService.signInWithGoogle(token).subscribe({
      next: (user) => console.log(user),
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }

  @HostListener('window:message', ['$event'])
  messageHandler($event: MessageEvent) {
    console.log($event);

    this.message = `${$event.data.message} (${$event.data.count})`;
    if ($event.data.count > 10) {
      window.close();
    }
  }

  login(): Observable<jose.JWTPayload> {
    return this.passkeysService.directPasskeyLogin().pipe(
      concatMap((token) => {
        localStorage.setItem('token', token);

        return from(jose.jwtVerify(token, this.passkeysService.JWKS, 
          { audience: 'wallet.flarex.io' }
        )).pipe(
          map(({ payload }) => payload)
        )
      })
    );
  }

  loginHandler() {
    this.login().subscribe({
      next: (payload) => this.name = payload.sub,
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
  }
}
