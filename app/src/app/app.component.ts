import { Component, HostListener } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Observable, catchError, concatMap, from, map } from 'rxjs';
import * as jose from 'jose';

import { PasskeysService } from './passkeys.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  title = 'app';
  name: string | undefined = 'Guest';
  message = '';

  constructor(
    private passkeysService: PasskeysService,
  ) {
    let login: Observable<jose.JWTPayload>;

    let token = localStorage.getItem('token');
    if (token == null) {
      login = this.login(); 
    } else {
      login = from(jose.jwtVerify(token, this.passkeysService.JWKS, 
        { audience: 'wallet.flarex.io' }
      )).pipe(
        map(({ payload }) => payload),
        catchError((e) => {
          console.error(e);

          return this.login();
        }),
      );
    }

    login.subscribe({
      next: (payload) => this.name = payload.sub,
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    })
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

  @HostListener('window:message', ['$event'])
  messageHandler($event: MessageEvent) {
    console.log($event);

    this.message = `${$event.data.message} (${$event.data.count})`;
    if ($event.data.count > 10) {
      window.close();
    }
  }
}
