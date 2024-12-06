import {
  AfterViewInit,
  Component,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import {
  NavigationEnd,
  Router,
  RouterModule,
  RouterOutlet,
} from '@angular/router';
import { filter, Subscription } from 'rxjs';
import { NotificationComponent } from './notification/notification.component';
import { NotificationService } from './notification/notification.service';
import { MessageService } from './services/message.service';

@Component({
  selector: 'app-root',
  imports: [RouterModule, RouterOutlet, NotificationComponent],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss',
})
export class AppComponent implements AfterViewInit, OnInit, OnDestroy {
  title = 'docker-image-puller';
  private subscription!: Subscription;
  showHomeButton = false;

  @ViewChild(NotificationComponent)
  notificationComponent!: NotificationComponent;

  constructor(
    private notificationService: NotificationService,
    private messageService: MessageService,
    private router: Router,
  ) {}

  ngOnInit() {
    // 订阅消息
    this.subscription = this.messageService.message$.subscribe((data) => {
      this.showNotification(data.message, data.isSuccess);
    });

    this.router.events
      .pipe(filter((event) => event instanceof NavigationEnd))
      .subscribe((event: NavigationEnd) => {
        const uri = event.urlAfterRedirects.split('?')[0];
        console.debug('url跳转', uri);

        // 检查当前路由是否是某个指定路由
        this.showHomeButton = this.isSpecificRoute(uri);
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

  isSpecificRoute(url: string): boolean {
    // 要显示home按钮的路由
    const specificRoutes = ['/detail'];
    return specificRoutes.includes(url);
  }
}
