import { Injectable, OnDestroy } from '@angular/core';
import { Subject, Subscription } from 'rxjs';
import { filter, map } from 'rxjs/operators';

interface Message<T> {
  topic: string;
  payload: T;
}

@Injectable({
  providedIn: 'root',
})
export class PubSubService implements OnDestroy {
  private messageSubject: Subject<Message<any>> = new Subject<Message<any>>();
  private subscriptions: Subscription[] = [];

  publish<T>(topic: string, payload: T): void {
    this.messageSubject.next({ topic, payload });
  }

  subscribe<T>(topic: string, callback: (payload: T) => void): Subscription {
    const subscription = this.messageSubject
      .asObservable()
      .pipe(
        filter((message) => message.topic === topic),
        map((message) => message.payload as T),
      )
      .subscribe({
        next(payload) {
          callback(payload);
        },
        error(e) {
          console.log(e);
        },
        complete() {},
      });

    this.subscriptions.push(subscription);
    return subscription;
  }

  ngOnDestroy(): void {
    this.subscriptions.forEach((subscription) => subscription.unsubscribe());
  }
}
