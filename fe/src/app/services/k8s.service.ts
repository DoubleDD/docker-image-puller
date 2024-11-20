import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class K8sService {
  private eventSource!: EventSource;

  constructor(private http: HttpClient) {}

  getPods(ns: string, deployment: string): Observable<string[]> {
    return this.http.get<string[]>(
      `/dip/api/dp/pods?ns=${ns}&dp=${deployment}`,
    );
  }

  getLogs(
    ns: string,
    podName: string,
    containerName: string,
  ): Observable<string> {
    return new Observable((observer) => {
      this.eventSource = new EventSource(
        `/dip/api/dp/logs?ns=${ns}&p=${podName}&c=${containerName}`,
      );
      this.eventSource.addEventListener('message', (event) => {
        observer.next(event.data);
      });
      this.eventSource.onerror = (error) => {
        observer.error(error);
      };

      return () => {
        // unsubscribed 时被调用
        this.eventSource.close();
      };
    });
  }
}
