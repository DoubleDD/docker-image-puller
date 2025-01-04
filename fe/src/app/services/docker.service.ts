import {
  HttpClient,
  HttpErrorResponse,
  HttpHeaders,
} from '@angular/common/http';
import { Injectable } from '@angular/core';
import { firstValueFrom, Observable, of, tap, throwError } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';
import { HttpService } from './http.service';

export interface Manifest {
  schemaVersion: number;
  mediaType: string;
  config: {
    digest: string;
    size: number;
  };
  layers: {
    mediaType: string;
    size: number;
    digest: string;
  }[];
}
export interface CachedManifest {
  manifest: Manifest;
  timestamp: number;
}

// 添加上传策略枚举
export enum UploadStrategy {
  NONE = 'none', // 不上传
  IMMEDIATE = 'immediate', // 在data事件中立即上传
  COMPLETE = 'complete', // 在complete事件中一次性上传
}

const proxy_server = 'http://localhost:7151';

@Injectable({
  providedIn: 'root',
})
export class DockerService {
  private eventSource!: EventSource;

  private readonly CACHE_KEY_PREFIX = 'docker_manifest_';
  private readonly CACHE_DURATION = 3 * 60 * 1000; // 3分钟的缓存时间（毫秒）
  private enableCache = false;

  constructor(
    private http: HttpClient,
    private httpService: HttpService,
  ) {
    const v = localStorage.getItem('cache.enable');
    if (v === 'true') {
      this.enableCache = true;
    }
  }

  getManifest(imageUrl: string): Observable<Manifest> {
    // 尝试从缓存获取
    const cachedData = this.getFromCache(imageUrl);
    if (cachedData) {
      return of(cachedData);
    }

    // 如果没有缓存或缓存已过期，则从服务器获取
    return this.http
      .get<Manifest>(
        `${proxy_server}/proxy/api/docker/manifest?image=${imageUrl}`,
      )
      .pipe(
        tap((manifest) => {
          // 保存到缓存
          this.saveToCache(imageUrl, manifest);
        }),
      );
  }

  private getFromCache(imageUrl: string): Manifest | null {
    if (!this.enableCache) {
      return null;
    }
    const cacheKey = this.CACHE_KEY_PREFIX + imageUrl;
    const cachedString = localStorage.getItem(cacheKey);

    if (!cachedString) {
      return null;
    }

    try {
      const cached: CachedManifest = JSON.parse(cachedString);

      // 检查缓存是否过期
      if (Date.now() - cached.timestamp > this.CACHE_DURATION) {
        localStorage.removeItem(cacheKey);
        return null;
      }

      return cached.manifest;
    } catch {
      localStorage.removeItem(cacheKey);
      return null;
    }
  }

  private saveToCache(imageUrl: string, manifest: Manifest): void {
    if (!this.enableCache) {
      return;
    }
    const cacheKey = this.CACHE_KEY_PREFIX + imageUrl;
    const cacheData: CachedManifest = {
      manifest,
      timestamp: Date.now(),
    };

    try {
      localStorage.setItem(cacheKey, JSON.stringify(cacheData));
    } catch (error) {
      // 如果localStorage已满，清除所有manifest缓存
      this.clearManifestCache();
      // 重试保存
      try {
        localStorage.setItem(cacheKey, JSON.stringify(cacheData));
      } catch {
        console.error('Failed to save manifest to cache');
      }
    }
  }

  private clearManifestCache(): void {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith(this.CACHE_KEY_PREFIX)) {
        localStorage.removeItem(key);
      }
    }
  }

  private createBlobDownloadStream(
    imageUrl: string,
    digest: string,
    size: number,
    onData?: (data: string) => void,
    uploadStrategy: UploadStrategy = UploadStrategy.IMMEDIATE, // 默认使用即时上传
  ): Observable<{ p: number; u: number; d: number }> {
    return new Observable((observer) => {
      let totalUploaded = 0;
      let chunkBuffer: string[] = [];
      let bufferSize = 0;
      let startCount = 0;
      let endCount = 0;
      let isComplete = false;

      const eventSource = new EventSource(
        `${proxy_server}/proxy/api/docker/blob/download?image=${imageUrl}&digest=${digest}&size=${size}`,
        { withCredentials: true },
      );

      eventSource.addEventListener('taskId', (event) => {
        //console.log('Download task created:', event.data);
      });

      eventSource.addEventListener('data', async (event) => {
        const chunk = JSON.parse(event.data);
        try {
          if (onData) {
            onData(chunk.data);
          }

          switch (uploadStrategy) {
            case UploadStrategy.NONE:
              break;
            case UploadStrategy.IMMEDIATE:
              // 收到数据后立马上传
              // 在网络不好的时候，一个请求要等很久才能结束，这里需要记录开始和结束的请求，只有两个相等时，才表明所有的请求都结束了
              // 开始的请求计数
              startCount++;
              const resp = await this.uploadChunckWithPool(
                digest,
                chunk.no,
                chunk.data,
                chunk.md5,
              );
              console.log('请求池返回：', resp);

              // 结束的请求计数
              endCount++;

              const chunkSize = Math.ceil((chunk.data.length * 3) / 4);
              totalUploaded += chunkSize;
              const uploadPercentage = (totalUploaded / size) * 100;
              observer.next({ p: -1, u: uploadPercentage, d: -1 });
              if (isComplete && startCount == endCount) {
                observer.complete();
              }
              break;
            case UploadStrategy.COMPLETE:
              // 完成后上传策略：收集数据
              chunkBuffer.push(event.data);
              bufferSize += Math.ceil((chunk.data.length * 3) / 4);
              break;
          }
        } catch (error) {
          console.error('Failed to process chunk:', error);
          observer.error('处理数据块失败');
          eventSource.close();
        }
      });

      eventSource.addEventListener('complete', (event) => {
        isComplete = true;
        const result = JSON.parse(event.data);
        if (result.digest === digest) {
          if (
            uploadStrategy === UploadStrategy.COMPLETE &&
            chunkBuffer.length > 0
          ) {
            // 完成后上传策略：一次性上传所有数据
            this.uploadChunck(digest, 0, chunkBuffer.join('')).subscribe({
              next: (_) => {
                observer.next({ p: 100, u: 100, d: result.size });
              },
              complete: () => {
                observer.complete();
              },
              error: (e) => {
                console.error('Failed to upload chunks:', e);
                observer.error('上传数据失败');
              },
            });
          } else {
            observer.next({ p: 100, u: 100, d: result.size });
            //console.log('startCount', startCount, 'endCount', endCount);
            if (startCount == endCount) {
              observer.complete();
            }
          }
        }
        eventSource.close();
      });

      eventSource.addEventListener('progress', (event) => {
        const progress = JSON.parse(event.data);
        if (progress.digest === digest) {
          observer.next({
            p: progress.percentage,
            u: -1,
            d: progress.downloaded,
          });
        }
      });

      eventSource.addEventListener('error', (event: any) => {
        observer.error(event.data || '下载失败');
        eventSource.close();
      });

      return () => eventSource.close();
    });
  }

  // 修改调用方法
  downloadLayer(
    imageUrl: string,
    digest: string,
    size: number,
  ): Observable<{ p: number; u: number; d: number }> {
    return this.createBlobDownloadStream(
      imageUrl,
      digest,
      size,
      undefined,
      UploadStrategy.IMMEDIATE,
    );
  }

  downloadConfig(
    imageUrl: string,
    digest: string,
    size: number,
  ): Observable<any> {
    const key = `config_${imageUrl}`;
    const cachedData = this.getFromCache(key);
    if (cachedData) {
      return of(cachedData);
    }

    let configData = '';
    return new Observable((observer) => {
      this.createBlobDownloadStream(
        imageUrl,
        digest,
        size,
        (data) => {
          configData += data;
        },
        UploadStrategy.NONE, // 配置文件使用完成后上传策略
      ).subscribe({
        complete: () => {
          try {
            const decodedData = atob(configData);
            const config = JSON.parse(decodedData);
            this.saveToCache(key, config);
            observer.next(config);
            observer.complete();
          } catch (error) {
            observer.error('Failed to parse config data');
          }
        },
        error: (error) => observer.error(error),
      });
    });
  }

  uploadChunckWithPool(
    digest: string,
    no: number,
    chunks: string,
    hash?: string,
  ): Promise<any> {
    return this.httpService.run<any>(() =>
      firstValueFrom(this.uploadChunck(digest, no, chunks, hash)),
    );
  }

  /**
   * 上传文件分片
   * @param digest 摘要信息
   * @param chunks 文件内容
   * @returns
   */
  uploadChunck(
    digest: string,
    no: number,
    chunks: string,
    hash?: string,
  ): Observable<any> {
    const index = no.toString().padStart(4, '0');
    return this.sendPreflightRequest(hash || '', index, digest).pipe(
      switchMap((flag) => {
        if (!flag) {
          return of(null); // 返回一个空值或错误信息
        }

        const formData = new FormData();
        formData.append('digest', digest);
        formData.append('no', index);
        formData.append('size', (chunks.length * 3) / 4 + '');
        formData.append('md5', hash || '');
        formData.append('chunk', chunks);

        return this.http.post(
          '/dip/api/docker/blob/chunk?digest=' + digest,
          formData,
        );
      }),
      catchError((error) => {
        if (error.message) {
          console.error('Error uploading chunk:', error);
        }
        return of(null); // 处理错误并返回一个空值或错误信息
      }),
    );
  }

  // 发送预检请求
  sendPreflightRequest(
    hash: string,
    index: string,
    digest: string,
  ): Observable<boolean> {
    const headers = new HttpHeaders({
      md5: hash || '',
      no: index,
      digest: digest,
    });

    return this.http
      .get('/dip/api/docker/blob/chunk/pre', { headers, observe: 'response' })
      .pipe(
        map((response) => {
          // 处理204 No Content状态码
          if (response.status === 204) {
            return true; // 表示可以继续上传文件
          }
          return false;
        }),
        catchError((error: HttpErrorResponse) => {
          // 处理403 Forbidden状态码
          if (error.status === 403) {
            return throwError(() => new Error(''));
          }
          // 处理其他错误
          return throwError(() => new Error('Preflight request failed'));
        }),
      );
  }
  /**
   * 合并镜像包
   * @param image 镜像地址
   * @param manifest
   * @param layers 层信息
   * @returns
   */
  mergeImage(
    image: string,
    manifest: Manifest | null,
    push: boolean,
  ): Observable<any> {
    const registry = localStorage.getItem('docker-registry') || '';
    return this.http.post('/dip/api/docker/merge', {
      image,
      manifest,
      push,
      registry,
    });
  }
  rollout(ns: string, deployment: string): Observable<any> {
    return this.http.get(`/dip/api/dp/rollout?ns=${ns}&name=${deployment}`);
  }

  checkLocalProxy(): Promise<any> {
    return firstValueFrom(
      this.http.get(proxy_server + '/proxy/status').pipe(
        catchError((e) => {
          console.log(e);
          return of(null);
        }),
      ),
    );
  }

  pullImage(newImage: string, oldImage: string): Observable<any> {
    return new Observable((observer) => {
      this.eventSource = new EventSource(
        `/dip/api/docker/pull?newImage=${newImage}&oldImage=${oldImage}`,
      );
      this.eventSource.addEventListener('message', (event) => {
        observer.next(event.data);
      });
      // 监听 done 事件
      this.eventSource.addEventListener('done', (event) => {
        console.log('Done:', event.data);
        this.eventSource.close(); // 关闭 EventSource 连接
      });
      this.eventSource.onerror = (error) => {
        console.log(error);
        observer.error(error);
      };

      return () => {
        // unsubscribed 时被调用
        this.eventSource.close();
      };
    });
  }
}
