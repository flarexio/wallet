import { Routes } from '@angular/router';

import { SignMessageComponent } from './sign-message/sign-message.component';
import { TokenTransferComponent } from './token-transfer/token-transfer.component';

export const routes: Routes = [
  { path: '', component: TokenTransferComponent },
  { path: 'tokens', component: TokenTransferComponent },
  { path: 'sign-message', component: SignMessageComponent },
];
