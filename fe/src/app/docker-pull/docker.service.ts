import { HttpClient, HttpEvent, HttpEventType } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { firstValueFrom, Observable, of, tap } from "rxjs";
import { Manifest } from "./docker-pull.component";




interface CachedManifest {
  manifest: Manifest;
  timestamp: number;
}


// 添加上传策略枚举
export enum UploadStrategy {
  NONE = 'none',  // 不上传
  IMMEDIATE = 'immediate',  // 在data事件中立即上传
  COMPLETE = 'complete'     // 在complete事件中一次性上传
}


const proxy_server = 'http://localhost:7152'


@Injectable({
  providedIn: "root",
})
export class DockerService {
  private readonly CACHE_KEY_PREFIX = 'docker_manifest_';
  private readonly CACHE_DURATION = 30 * 60 * 1000; // 30分钟的缓存时间（毫秒）

  constructor(private http: HttpClient) { }

  getManifest(imageUrl: string): Observable<Manifest> {
    // 尝试从缓存获取
    const cachedData = this.getFromCache(imageUrl);
    if (cachedData) {
      return of(cachedData);
    }

    // 如果没有缓存或缓存已过期，则从服务器获取
    return this.http.get<Manifest>(
      `${proxy_server}/dip/api/docker/manifest?image=${imageUrl}`
    ).pipe(
      tap(manifest => {
        // 保存到缓存
        this.saveToCache(imageUrl, manifest);
      })
    );
  }

  private getFromCache(imageUrl: string): Manifest | null {
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
    const cacheKey = this.CACHE_KEY_PREFIX + imageUrl;
    const cacheData: CachedManifest = {
      manifest,
      timestamp: Date.now()
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
    uploadStrategy: UploadStrategy = UploadStrategy.IMMEDIATE  // 默认使用即时上传
  ): Observable<{ p: number, u: number, d: number }> {
    return new Observable(observer => {
      let totalUploaded = 0;
      let chunkBuffer: string[] = [];
      let bufferSize = 0;

      const eventSource = new EventSource(
        `${proxy_server}/dip/api/docker/blob/download?image=${imageUrl}&digest=${digest}&size=${size}`,
        { withCredentials: true }
      );

      eventSource.addEventListener('taskId', (event) => {
        console.log('Download task created:', event.data);
      });

      eventSource.addEventListener('data', (event) => {
        try {
          if (onData) {
            console.log(event.data);

            onData(event.data);
          }

          switch (uploadStrategy) {
            case UploadStrategy.NONE:
              break;
            case UploadStrategy.IMMEDIATE:
              // 即时上传策略
              const chunkSize = Math.ceil(event.data.length * 3 / 4);
              firstValueFrom(this.uploadChunck(digest, event.data))
                .catch(error => {
                  console.error('Failed to upload chunk:', error);
                });

              totalUploaded += chunkSize;
              const uploadPercentage = (totalUploaded / size) * 100;
              observer.next({ p: -1, u: uploadPercentage, d: -1 });
              break;
            case UploadStrategy.COMPLETE:
              // 完成后上传策略：收集数据
              chunkBuffer.push(event.data);
              bufferSize += Math.ceil(event.data.length * 3 / 4);
              break;
          }

        } catch (error) {
          console.error('Failed to process chunk:', error);
          observer.error('处理数据块失败');
          eventSource.close();
        }
      });

      eventSource.addEventListener('complete', (event) => {
        const result = JSON.parse(event.data);
        if (result.digest === digest) {
          if (uploadStrategy === UploadStrategy.COMPLETE && chunkBuffer.length > 0) {
            // 完成后上传策略：一次性上传所有数据
            firstValueFrom(this.uploadChunck(digest, chunkBuffer.join("")))
              .then(() => {
                observer.next({ p: 100, u: 100, d: result.size });
                observer.complete();
              })
              .catch(error => {
                console.error('Failed to upload chunks:', error);
                observer.error('上传数据失败');
              });
          } else {
            observer.next({ p: 100, u: 100, d: result.size });
            observer.complete();
          }
        }
        eventSource.close();
      });

      eventSource.addEventListener('progress', (event) => {
        const progress = JSON.parse(event.data);
        if (progress.digest === digest) {
          observer.next({ p: progress.percentage, u: -1, d: progress.downloaded });
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
  downloadLayer(imageUrl: string, digest: string, size: number): Observable<{ p: number, u: number, d: number }> {
    return this.createBlobDownloadStream(imageUrl, digest, size, undefined, UploadStrategy.IMMEDIATE);
  }



  downloadConfig(imageUrl: string, digest: string, size: number): Observable<any> {
    const key = `config_${imageUrl}`
    const cachedData = this.getFromCache(key);
    if (cachedData) {
      return of(cachedData);
    }

    let configData = '';
    return new Observable(observer => {
      this.createBlobDownloadStream(
        imageUrl,
        digest,
        size,
        (data) => { configData += data },
        UploadStrategy.NONE  // 配置文件使用完成后上传策略
      ).subscribe({
        complete: () => {
          try {
            const decodedData = atob(configData);
            const config = JSON.parse(decodedData);
            this.saveToCache(key, config)
            observer.next(config);
            observer.complete();
          } catch (error) {
            observer.error('Failed to parse config data');
          }
        },
        error: (error) => observer.error(error)
      });
    });
  }

  /**
   * 上传文件分片
   * @param digest 摘要信息
   * @param chunks 文件内容
   * @returns
   */
  uploadChunck(digest: string, chunks: string): Observable<any> {
    const formData = new FormData();
    formData.append('digest', digest);
    formData.append('chunk', chunks);
    return this.http.post('/dip/api/docker/blob/chunk', formData);
  }

  /**
   * 合并镜像包
   * @param image 镜像地址
   * @param manifest
   * @param layers 层信息
   * @returns
   */
  mergeImage(image: string, manifest: Manifest): Observable<any> {
    return this.http.post("/dip/api/docker/merge", { image, manifest });
  }
}
