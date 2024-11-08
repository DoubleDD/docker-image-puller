import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'stringLength',
  standalone: true,
})
export class stringLengthPipe implements PipeTransform {
  transform(str: string): string {
    if (str.length > 20) {
      return str.substr(0, 20) + '...';
    }
    return str;
  }
}
