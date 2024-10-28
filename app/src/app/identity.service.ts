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
export class IdentityService {
  public JWKS = jose.createRemoteJWKSet(new URL(`${env.PASSKEYS_BASEURL}/.well-known/jwks.json`));

  private baseURL = env.FLAREX_IDENTITY_BASEURL + '/identity/v1';

  constructor(
    private http: HttpClient,
  ) { }

  signin(provider: string, token: string): Observable<User> {
    const params = {
      'provider': provider,
      'credential': token,
    };

    return this.http.patch(`${this.baseURL}/signin`, params).pipe(
      map((raw) => Object.assign(new User(), raw)),
    );
  }

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

export class User {
  public id: string = '';
  public username: string = '';
  public name: string = '';
  public email: string = '';
  public status: number = 0;
  public avatar: string = '';

  private _accounts: SocialAccount[] = [];
  private _token: Token | undefined;

  public get accounts(): SocialAccount[] {
      return this._accounts;
  }
  public set accounts(value: SocialAccount[]) {
      const accounts: SocialAccount[] = new Array();
      if (value != null) {
          for (const account of value) {
              accounts.push(Object.assign(new SocialAccount(), account));
          }
      }

      this._accounts = accounts;
  }

  public get token(): Token | undefined {
      return this._token;
  }
  public set token(value: Token | undefined) {
      if (value == undefined) {
          this._token = undefined;
      } else {
          this._token = Object.assign(new Token(), value);
      }
  }
}

export class SocialAccount {
  public social_id: string = '';
  public social_provider: string = '';
}

export class Token {
  public token: string = '';
  public expired_at: string = '';
}
