import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';

import { WalletMessage, WalletMessageType, SignMessagePayload } from '@flarex/wallet-adapter';
import { v4 as uuid } from 'uuid';
import * as base58 from 'bs58';

import { WalletService } from '../wallet.service';

@Component({
  selector: 'app-sign-message',
  standalone: true,
  imports: [
    FormsModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatCardModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
  ],
  templateUrl: './sign-message.component.html',
  styleUrl: './sign-message.component.scss'
})
export class SignMessageComponent {
  msg: WalletMessage | undefined;
  message = '';
  signature = '';
  displayedMessage = '';

  constructor(
    private router: Router,
    private walletService: WalletService,
  ) {
    const nav = this.router.getCurrentNavigation();
    const state = nav?.extras.state;
    if (state == undefined) return;

    const msg = state['msg'] as WalletMessage;

    if (msg.type != WalletMessageType.SIGN_MESSAGE) return;

    this.msg = msg;

    const payload = msg.payload as SignMessagePayload;
    const msgStr = new TextDecoder().decode(payload.msg);
    this.message = msgStr;
  }

  signMessage(message: string) {
    if (this.msg == undefined) {
      const tid = uuid();
      const msg = new TextEncoder().encode(message);

      this.walletService.signMessage(tid, msg).subscribe({
        next: (result) => this.signature = result.sig,
        error: (err) => console.error(err),
        complete: () => console.log('complete'),
      });
    } else {
      this.walletService.messageHandler(this.msg).subscribe({
        next: (resp) => {
          const payload = resp.payload as SignMessagePayload;
          if (payload.sig) {
            const sig = base58.encode(payload.sig);
            this.signature = sig;
          }

          this.walletService.sendResponse(resp);
        },
        error: (err) => console.error(err),
        complete: () => window.close(),
      });
    }
  }

  onDisplayChange(display: string) {
    const msg = this.msg;
    if (msg == undefined) return;

    const payload = msg.payload as SignMessagePayload;
    const msgStr = new TextDecoder().decode(payload.msg);
    console.log(msgStr);

    switch (display) {
      case 'json':
        const msg = JSON.parse(msgStr);
        this.message = JSON.stringify(msg, null, 2);
        break;

      case 'hex':
        this.message = Array.from(msgStr)
          .map(c => c.charCodeAt(0).toString(16).padStart(2, '0'))
          .join(' ');
        break;

      case 'base64':
        this.message = btoa(msgStr);
        break;

      default:
        this.message = msgStr;
        break;
    }
  }
}
