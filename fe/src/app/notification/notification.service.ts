import { Injectable } from '@angular/core';
import { NotificationComponent } from './notification.component';

@Injectable({
  providedIn: 'root',
})
export class NotificationService {
  private notificationComponent: NotificationComponent | null = null;

  register(notificationComponent: NotificationComponent) {
    this.notificationComponent = notificationComponent;
  }

  show(message: string, isSuccess: boolean) {
    this.notificationComponent?.showNotification(message, isSuccess);
  }
}
