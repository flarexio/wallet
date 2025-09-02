import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Observable, map, catchError, of, timeout } from 'rxjs';

export interface PythPriceData {
  id: string;
  price: {
    price: string;
    conf: string;
    expo: number;
    publish_time: number;
  };
  ema_price: {
    price: string;
    conf: string;
    expo: number;
    publish_time: number;
  };
}

export interface PythResponse {
  parsed: PythPriceData[];
}

export interface SolanaPrice {
  price: number;
  confidence: number;
  lastUpdated: Date;
}

@Injectable({
  providedIn: 'root'
})
export class PythService {
  private readonly PYTH_API_URL = 'https://hermes.pyth.network/v2/updates/price/latest';
  private readonly SOL_USD_PRICE_ID = 'ef0d8b6fda2ceba41da15d4095d1da392a0d2f8ed0c6c7bc0f4cfac8c280b56d';

  constructor(private http: HttpClient) {}

  /**
   * 獲取 Solana/USD 最新價格
   */
  getSolanaPrice(): Observable<SolanaPrice | null> {
    const url = `${this.PYTH_API_URL}?parsed=true&ids[]=${this.SOL_USD_PRICE_ID}`;
    console.log('Fetching price from:', url); // 添加除錯資訊

    return this.http.get<PythResponse>(url).pipe(
      timeout(10000), // 10秒超時
      map(response => {
        console.log('Pyth API response:', response); // 添加除錯資訊
        
        if (!response.parsed || response.parsed.length === 0) {
          console.warn('No price data received from Pyth');
          return null;
        }

        const priceData = response.parsed[0];
        const price = this.convertPythPrice(priceData.price.price, priceData.price.expo);
        const confidence = this.convertPythPrice(priceData.price.conf, priceData.price.expo);
        const lastUpdated = new Date(priceData.price.publish_time * 1000);

        const result = {
          price,
          confidence,
          lastUpdated
        } as SolanaPrice;

        console.log('Processed price:', result); // 添加除錯資訊
        return result;
      }),
      catchError((error: HttpErrorResponse) => {
        console.error('Error fetching Solana price from Pyth:', error);
        // 返回 null 而不是拋出錯誤，這樣不會中斷整個資料流
        return of(null);
      })
    );
  }

  /**
   * 轉換 Pyth 價格格式（處理 expo）
   */
  private convertPythPrice(price: string, expo: number): number {
    const priceNum = parseInt(price, 10);
    if (isNaN(priceNum)) {
      console.warn('Invalid price format:', price);
      return 0;
    }
    const result = priceNum * Math.pow(10, expo);
    console.log(`Converting price: ${price} * 10^${expo} = ${result}`); // 添加除錯資訊
    return result;
  }

  /**
   * 格式化價格顯示
   */
  formatPrice(price: number): string {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(price);
  }

  /**
   * 計算 SOL 數量對應的 USD 價值
   */
  calculateUSDValue(solAmount: number, price: number): number {
    if (!solAmount || !price || isNaN(solAmount) || isNaN(price)) {
      return 0;
    }
    return solAmount * price;
  }
}
