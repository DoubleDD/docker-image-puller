import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-notification',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './notification.component.html',
  styleUrl: './notification.component.scss',
})
export class NotificationComponent {
  @Input() message: string = 'Ok!';
  visible = false;
  isSuccess = true;

  showNotification(message: string, isSuccess: boolean) {
    this.message = message;
    this.visible = true;
    this.isSuccess = isSuccess;

    // 自动在3秒后隐藏
    setTimeout(() => {
      this.visible = false;
    }, 3000);
  }
}
