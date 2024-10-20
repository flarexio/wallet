import { Component, HostListener } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { concatMap, from } from 'rxjs';
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
  name: string | undefined = '';
  message = '';

  constructor(
    private passkeysService: PasskeysService,
  ) {
    this.passkeysService.directPasskeyLogin().pipe(
      concatMap((token) => from(
        jose.jwtVerify(token, this.passkeysService.JWKS, {
          audience: 'wallet.flarex.io',
        })
      ))
    ).subscribe({
      next: ({ payload }) => this.name = payload.sub,
      error: (err) => console.error(err),
      complete: () => console.log('complete'),
    });
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
