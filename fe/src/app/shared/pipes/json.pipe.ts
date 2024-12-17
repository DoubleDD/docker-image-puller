import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'json',
  standalone: true,
})
export class JsonPipe implements PipeTransform {
  transform(str: string): string {
    const result = JSON.stringify(str, null, 4);
    return result;
  }
}
