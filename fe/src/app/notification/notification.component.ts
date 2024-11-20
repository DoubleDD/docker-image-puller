import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';

@Component({
    selector: 'app-notification',
    imports: [CommonModule],
    templateUrl: './notification.component.html',
    styleUrl: './notification.component.scss'
})
export class NotificationComponent {
  @Input() message: string = 'Ok!';
  visible = false;
  isSuccess = true;

  showNotification(message: string, isSuccess: boolean) {
    this.message = message;
    this.visible = true;
    this.isSuccess = isSuccess;

    // 自动在9秒后隐藏
    setTimeout(() => {
      this.visible = false;
    }, 9000);
  }
}
