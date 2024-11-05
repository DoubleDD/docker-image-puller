import { HttpClient, HttpEvent, HttpEventType } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { Observable, of, tap } from "rxjs";
import { Manifest } from "./docker-pull.component";

const proxy_server = 'http://localhost:7152'

interface CachedManifest {
  manifest: Manifest;
  timestamp: number;
}

@Injectable({
  providedIn: "root",
})
export class DockerService {
  private readonly CACHE_KEY_PREFIX = 'docker_manifest_';
  private readonly CACHE_DURATION =  30 * 60 * 1000; // 30分钟的缓存时间（毫秒）

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
    onData?: (data: string) => void
  ): Observable<{ p: number, u: number, d: number }> {
    return new Observable(observer => {
      let totalUploaded = 0;

      const eventSource = new EventSource(
        `${proxy_server}/dip/api/docker/blob/download?image=${imageUrl}&digest=${digest}&size=${size}`,
        { withCredentials: true }
      );

      eventSource.addEventListener('taskId', (event) => {
        console.log('Download task created:', event.data);
      });

      eventSource.addEventListener('data', async (event) => {
        try {
          // 如果有自定义数据处理函数，先处理数据
          if (onData) {
            onData(event.data);
          }

          const formData = new FormData();
          formData.append('digest', digest);
          formData.append('chunk', event.data);

          const chunkSize = Math.ceil(event.data.length * 3 / 4);
          await this.http.post('/dip/api/docker/blob/chunk', formData).toPromise();

          totalUploaded += chunkSize;
          const uploadPercentage = (totalUploaded / size) * 100;
          observer.next({ p: -1, u: uploadPercentage, d: -1 });

        } catch (error) {
          console.error('Failed to upload chunk:', error);
          observer.error('上传数据块失败');
          eventSource.close();
        }
      });

      eventSource.addEventListener('progress', (event) => {
        const progress = JSON.parse(event.data);
        if (progress.digest === digest) {
          observer.next({ p: progress.percentage, u: -1, d: progress.downloaded });
        }
      });

      eventSource.addEventListener('complete', (event) => {
        const result = JSON.parse(event.data);
        if (result.digest === digest) {
          observer.next({ p: 100, u: 100, d: result.size });
          observer.complete();
        }
        eventSource.close();
      });

      eventSource.addEventListener('error', (event: any) => {
        observer.error(event.data || '下载失败');
        eventSource.close();
      });

      return () => eventSource.close();
    });
  }

  downloadLayer(imageUrl: string, digest: string, size: number): Observable<{ p: number, u: number, d: number }> {
    return this.createBlobDownloadStream(imageUrl, digest, size);
  }

  mergeImage(image: string, manifest: Manifest, layers: string[]): Observable<any> {
    return this.http.post("/dip/api/docker/merge", { image, manifest, layers });
  }

  downloadConfig(imageUrl: string, digest: string, size: number): Observable<any> {
    let configData = '';

    return new Observable(observer => {
      this.createBlobDownloadStream(
        imageUrl,
        digest,
        size,
        (data) => { configData += data }
      ).subscribe({
        complete: () => {
          try {
            const decodedData = atob(configData);
            const config = JSON.parse(decodedData);
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
}
