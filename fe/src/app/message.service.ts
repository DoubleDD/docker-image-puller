import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class MessageService {
  constructor() {}

  private messageSubject = new Subject<{
    message: string;
    isSuccess: boolean;
  }>();

  // Observable 用于订阅消息
  message$ = this.messageSubject.asObservable();

  // 发布消息的方法
  publish(message: string, isSuccess: boolean) {
    this.messageSubject.next({ message, isSuccess });
  }
}
