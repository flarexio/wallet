import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable, concatMap, map } from 'rxjs';
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

  private _currentUser: User | undefined;
  private _currentToken: Token | undefined;

  private _userChangeSubject = new BehaviorSubject<User | undefined>(undefined);

  public userChange = this._userChangeSubject.asObservable();

  constructor(
    private http: HttpClient,
  ) { }

  signin(provider: string, token: string): Observable<User> {
    const params = {
      'provider': provider,
      'credential': token,
    };

    return this.http.patch(`${this.baseURL}/signin`, params).pipe(
      map((raw: any) => {
        const token = Object.assign(new Token(), raw.token);
        const user = Object.assign(new User(), raw.user);

        this.currentToken = token;
        this.currentUser = user;
        return user;
      }),
    );
  }

  registerPasskey(): Observable<User> {
    if (this.currentUser == undefined) {
      throw new Error('user not found');
    }

    const user = this.currentUser.username;

    if (this.currentToken == undefined) {
      throw new Error('token not found');
    }

    const headers = { 'Authorization': `Bearer ${this.currentToken.token}` };

    return this.http.post(`${this.baseURL}/users/${user}/passkeys/register`, null, { headers }).pipe(
      concatMap((opts) => create(opts as CredentialCreationOptionsJSON)),
      concatMap((credential) => this.http.post(`${this.baseURL}/passkeys/registration`, credential)),
      concatMap((token) => this.http.put(`${this.baseURL}/users/${user}/socials`, {
        'credential': token,
        'provider': 'passkeys',
      }, { headers })),
      map((raw) => {
        const user = Object.assign(new User(), raw);

        this.currentUser = user;
        return user;
      }),
    );
  }

  directPasskeyLogin(): Observable<string> {
    return this.http.post(`${env.PASSKEYS_BASEURL}/login/initialize`, null).pipe(
      concatMap((opts) => get(opts as CredentialRequestOptionsJSON)),
      concatMap((credential) => this.http.post(`${env.PASSKEYS_BASEURL}/login/finalize`, credential)),
      map((result: any) => result.token),
    );
  }

  public get currentUser(): User | undefined {
    return this._currentUser;
  }
  public set currentUser(user: User | undefined) {
    this._currentUser = user;
    this._userChangeSubject.next(user);
  }

  public get currentToken(): Token | undefined {
    return this._currentToken;
  }
  public set currentToken(token: Token | undefined) {
    this._currentToken = token;
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
}

export class SocialAccount {
  public social_id: string = '';
  public social_provider: string = '';
}

export class Token {
  public token: string = '';
  public expired_at: string = '';
}
