import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable, concatMap, from, map } from 'rxjs';
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
  private _currentPasskeyUserID: string | undefined;

  private _userChangeSubject = new BehaviorSubject<User | undefined>(undefined);

  userChange = this._userChangeSubject.asObservable();

  constructor(
    private http: HttpClient,
  ) {
    const passkeyUserID = localStorage.getItem('passkeys-user-id');
    if (passkeyUserID != undefined) {
      this._currentPasskeyUserID = passkeyUserID;
    }
  }

  signin(provider: string, token: string): Observable<SigninResult> {
    const params = {
      'provider': provider,
      'credential': token,
    };

    return this.http.patch(`${this.baseURL}/signin`, params).pipe(
      map((raw: any) => {
        const token = raw.token as Token;
        const user = raw.user as User;

        this.currentToken = token;
        this.currentUser = user;
        return { user, token };
      }),
    );
  }

  getUserFromToken(tokenStr: string): Observable<SigninResult> {
    const headers = { 'Authorization': `Bearer ${tokenStr}` };

    return this.http.get(`${this.baseURL}/token/user`, { headers }).pipe(
      map((raw) => {
        const user = raw as User;
        const token = { token: tokenStr, expired_at: '' };

        this.currentToken = token;
        this.currentUser = user;
        return { user, token };
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

    const token = this.currentToken.token;
    const headers = { 'Authorization': `Bearer ${token}` };

    return this.http.post(`${this.baseURL}/users/${user}/passkeys/register`, null, { headers }).pipe(
      concatMap((opts) => create(opts as CredentialCreationOptionsJSON)),
      concatMap((credential) => this.http.post(`${this.baseURL}/passkeys/registration`, credential)),
      concatMap((token) => this.http.put(`${this.baseURL}/users/${user}/socials`, {
        'credential': token,
        'provider': 'passkeys',
      }, { headers })),
      map((raw: any) => {
        const user = raw.user as User;

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

  verifyPasskeyToken(token: string): Observable<jose.JWTPayload> {
    return from(jose.jwtVerify(token, this.JWKS, {
      audience: 'wallet.flarex.io'
    })).pipe(
      map(({ payload }) => { 
        this.currentPasskeyUserID = payload.sub;

        return payload;
      })
    )
  }

  logout() {
    localStorage.removeItem('passkeys-user-id');
    this.currentUser = undefined;
    this.currentToken = undefined;
    this.currentPasskeyUserID = undefined;
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

  public get currentPasskeyUserID(): string | undefined {
    return this._currentPasskeyUserID;
  }
  public set currentPasskeyUserID(userID: string | undefined) {
    this._currentPasskeyUserID = userID;

    if (userID != undefined) {
      localStorage.setItem('passkeys-user-id', userID);
    }
  }
}

export interface SigninResult {
  user: User;
  token: Token;
}

export interface User {
  id: string;
  username: string;
  name: string;
  email: string;
  status: number;
  accounts: SocialAccount[];
  avatar: string;
}

export interface SocialAccount {
  social_id: string;
  social_provider: string;
}

export interface Token {
  token: string;
  expired_at: string;
}
