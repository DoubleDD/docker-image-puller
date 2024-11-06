import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'mathFloor',
  standalone: true
})
export class MathFloorPipe implements PipeTransform {
  transform(percent: number): number {
    return Math.floor(percent);
  }
}
