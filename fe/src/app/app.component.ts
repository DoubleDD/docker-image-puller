import {
  AfterViewInit,
  Component,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import { RouterLink, RouterOutlet } from '@angular/router';
import { NotificationComponent } from './notification/notification.component';
import { NotificationService } from './notification/notification.service';
import { MessageService } from './message.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterLink, RouterOutlet, NotificationComponent],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss',
})
export class AppComponent implements AfterViewInit, OnInit, OnDestroy {
  title = 'docker-image-puller';
  private subscription!: Subscription;

  @ViewChild(NotificationComponent)
  notificationComponent!: NotificationComponent;

  constructor(
    private notificationService: NotificationService,
    private messageService: MessageService,
  ) {}

  ngOnInit() {
    // 订阅消息
    this.subscription = this.messageService.message$.subscribe((data) => {
      this.showNotification(data.message, data.isSuccess);
    });
  }

  ngOnDestroy() {
    // 取消订阅，防止内存泄漏
    this.subscription.unsubscribe();
  }
  ngAfterViewInit() {
    this.notificationService.register(this.notificationComponent);
  }

  showNotification(message: string, isSuccess: boolean) {
    this.notificationService.show(message, isSuccess);
  }
}
