import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class RequestQueueService {
  private taskQueue: (() => Promise<any>)[] = [];
  private isProcessing = false;

  constructor(private http: HttpClient) {}

  // 任务调度器，添加任务到队列
  public scheduleTask(task: () => Promise<any>): void {
    this.taskQueue.push(task);
    this.processQueue();
  }

  // 处理队列中的任务
  private async processQueue(): Promise<void> {
    // 如果已经在处理任务或任务队列为空，则直接返回
    if (this.isProcessing || this.taskQueue.length === 0) {
      return;
    }

    this.isProcessing = true;

    while (this.taskQueue.length > 0) {
      const currentTask = this.taskQueue.shift();
      if (currentTask) {
        try {
          await currentTask(); // 按顺序执行任务
        } catch (error) {
          console.error("任务执行失败", error);
        }
      }
    }

    this.isProcessing = false; // 所有任务处理完后重置标志
  }


  public get(url: string, options?: any): Promise<any> {
    return new Promise((resolve, reject) => {
      this.scheduleTask(() =>
        firstValueFrom(this.http.get(url, options)).then(resolve).catch(reject)
      );
    });
  }

  public post(url: string, options?: any): Promise<any> {
    return new Promise((resolve, reject) => {
      this.scheduleTask(() =>
        firstValueFrom(this.http.post(url, options)).then(resolve).catch(reject)
      );
    });
  }
}
