import { Routes } from '@angular/router';

import { ConnectedSitesComponent } from './connected-sites/connected-sites.component';
import { SignMessageComponent } from './sign-message/sign-message.component';
import { TokenTransferComponent } from './token-transfer/token-transfer.component';

export const routes: Routes = [
  { path: '', component: TokenTransferComponent },
  { path: 'tokens', component: TokenTransferComponent },
  { path: 'sign-message', component: SignMessageComponent },
  { path: 'connected-sites', component: ConnectedSitesComponent },
];
