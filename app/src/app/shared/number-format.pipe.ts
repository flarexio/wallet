import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'numberFormat',
  standalone: true
})
export class NumberFormatPipe implements PipeTransform {

  transform(value: number, currency: string = '$', digits: number = 2): string {
    if (value >= 1_000_000_000_000) {
      return `${currency}${(value / 1_000_000_000_000).toFixed(digits)}T`;
    } else if (value >= 1_000_000_000) {
      return `${currency}${(value / 1_000_000_000).toFixed(digits)}B`;
    } else if (value >= 1_000_000) {
      return `${currency}${(value / 1_000_000).toFixed(digits)}M`;
    } else {
      const formatted = value.toLocaleString('en-US', {
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      });

      return `${currency}${formatted}`;
    }
  }
}
