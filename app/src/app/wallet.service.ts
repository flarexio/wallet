import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';

import { environment as env } from '../environments/environment';
import { IdentityService } from './identity.service';

@Injectable({
  providedIn: 'root'
})
export class WalletService {
  private baseURL = env.FLAREX_WALLET_BASEURL;

  constructor(
    private http: HttpClient,
    private identity: IdentityService,
  ) { }

  wallet(): Observable<string> {
    const user = this.identity.currentUser?.username;
    if (user == undefined) {
      throw new Error('user not found');
    }

    const token = this.identity.currentToken?.token;
    if (token == undefined) {
      throw new Error('token not found');
    }

    return this.http.get<string>(`${this.baseURL}/wallets/${user}`, 
      { headers: { Authorization: `Bearer ${token}` } });
  }
}
