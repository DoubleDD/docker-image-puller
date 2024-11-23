import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class HttpService {
  private requestPool: RequestPool;

  constructor() {
    this.requestPool = new RequestPool(5);
  }

  run<T>(task: () => Promise<T>): Promise<T> {
    return this.requestPool.add(task);
  }
}

class RequestPool {
  private queue: (() => Promise<any>)[] = [];
  private activeCount = 0;
  private readonly concurrency: number;

  constructor(concurrency: number) {
    this.concurrency = concurrency;
  }

  // 添加任务到队列
  add(task: () => Promise<any>): Promise<any> {
    return new Promise((resolve, reject) => {
      const runTask = async () => {
        try {
          this.activeCount++;
          const result = await task();
          resolve(result);
        } catch (error) {
          reject(error);
        } finally {
          this.activeCount--;
          this.next(); // 任务完成后，执行下一个任务
        }
      };

      this.queue.push(runTask);
      this.next(); // 检查是否可以执行任务
    });
  }

  // 检查是否可以执行任务
  private next() {
    if (this.activeCount < this.concurrency && this.queue.length > 0) {
      const nextTask = this.queue.shift();
      if (nextTask) {
        nextTask();
      }
    }
  }
}
