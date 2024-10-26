import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';

import { environment as env } from '../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class IdentityService {
  private baseURL = env.FLAREX_IDENTITY_BASEURL + '/identity/v1';

  constructor(
    private http: HttpClient,
  ) { }

  signInWithGoogle(token: string): Observable<User> {
    const params = {
      'provider': 'google',
      'credential': token,
    };

    return this.http.patch(`${this.baseURL}/signin`, params).pipe(
      map((raw) => Object.assign(new Result<User>(), raw)),
      map((result) => {
        if (result.status != 'success') {
          throw new Error(result.msg);
        }

        return Object.assign(new User(), result.data);
      })
    );
  }
}

export class Result<T> {
  public status: string = '';
  public msg: string = '';
  public data: T | undefined;
  public time: string = '';
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
