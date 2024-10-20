import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, concatMap, map } from 'rxjs';
import * as jose from 'jose';

import { 
  CredentialCreationOptionsJSON, CredentialRequestOptionsJSON, 
  create, get, 
} from "@github/webauthn-json";

import { environment as env } from '../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class PasskeysService {
  public JWKS = jose.createRemoteJWKSet(new URL(`${env.PASSKEYS_BASEURL}/.well-known/jwks.json`));

  private baseURL = env.FLAREX_WALLET_BASEURL;

  constructor(
    private http: HttpClient,
  ) { }

  registerPasskey(user_id: string, username: string): Observable<string> {
    return this.http.post(`${this.baseURL}/passkeys/registration/initialize`, { user_id, username }).pipe(
      concatMap((opts) => create(opts as CredentialCreationOptionsJSON)),
      concatMap((credential) => this.http.post(`${this.baseURL}/passkeys/registration/finalize`, credential)),
      map((result: any) => result.token),
    );
  }

  loginWithPasskey(user_id: string): Observable<string> {
    return this.http.post(`${this.baseURL}/passkeys/login/initialize`, { user_id }).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.post(`${this.baseURL}/passkeys/login/finalize`, credential)),
      map((result: any) => result.token),
    );
  }

  directPasskeyLogin(): Observable<string> {
    return this.http.post(`${env.PASSKEYS_BASEURL}/login/initialize`, null).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.post(`${env.PASSKEYS_BASEURL}/login/finalize`, credential)),
      map((result: any) => result.token),
    );
  }
}
