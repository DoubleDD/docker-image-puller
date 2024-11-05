import { HttpClient, HttpEvent, HttpEventType } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { Observable, of, tap } from "rxjs";
import { map } from "rxjs/operators";
import { Manifest } from "./docker-pull.component";

interface CachedManifest {
  manifest: Manifest;
  timestamp: number;
}

interface DownloadProgress {
  taskId: string;
  status: string;
  progress: {
    [digest: string]: {
      size: number;
      downloaded: number;
      percentage: number;
      status: string;
    };
  };
  error?: string;
}

@Injectable({
  providedIn: "root",
})
export class DockerService {
  private readonly CACHE_KEY_PREFIX = 'docker_manifest_';
  private readonly CACHE_DURATION = 24 * 60 * 60 * 1000; // 24小时的缓存时间（毫秒）

  constructor(private http: HttpClient) {}

  getManifest(imageUrl: string): Observable<Manifest> {
    // 尝试从缓存获取
    const cachedData = this.getFromCache(imageUrl);
    if (cachedData) {
      return of(cachedData);
    }

    // 如果没有缓存或缓存已过期，则从服务器获取
    return this.http.get<Manifest>(
      `/dip/api/docker/manifest?image=${imageUrl}`
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

  downloadLayer(imageUrl: string, digest: string, size: number): Observable<number> {
    return new Observable<number>(observer => {
      // 首先发起下载请求
      this.http.post<{ taskId: string }>('/dip/api/docker/blob/download', {
        image: imageUrl,
        digest: digest,
        size: size
      }).subscribe({
        next: (response) => {
          const taskId = response.taskId;

          // 开始轮询进度
          const pollInterval = setInterval(() => {
            this.http.get<DownloadProgress>(`/dip/api/docker/blob/progress?taskId=${taskId}`)
              .subscribe({
                next: (progress) => {
                  if (progress.error) {
                    observer.error(progress.error);
                    clearInterval(pollInterval);
                    return;
                  }

                  const layerProgress = progress.progress[digest];
                  if (layerProgress) {
                    observer.next(layerProgress.percentage);
                  }

                  // 检查是否完成
                  if (progress.status === 'completed') {
                    observer.next(100);
                    observer.complete();
                    clearInterval(pollInterval);
                  } else if (progress.status === 'failed') {
                    observer.error('Download failed');
                    clearInterval(pollInterval);
                  }
                },
                error: (err) => {
                  observer.error(err);
                  clearInterval(pollInterval);
                }
              });
          }, 1000); // 每秒轮询一次

          // 清理函数
          return () => {
            clearInterval(pollInterval);
          };
        },
        error: (err) => observer.error(err)
      });
    });
  }

  uploadLayer(blob: Blob, digest: string): Observable<any> {
    const formData = new FormData();
    formData.append("layer", blob);
    formData.append("digest", digest);

    return this.http
      .post("/dip/api/docker/upload", formData, {
        reportProgress: true,
        observe: "events",
      })
      .pipe(
        map((event: HttpEvent<any>) => {
          if (event.type === HttpEventType.UploadProgress && event.total) {
            const progress = Math.round((event.loaded / event.total) * 100);
            // You'll need to implement a way to update the progress in the component
          }
          if (event.type === HttpEventType.Response) {
            return event.body;
          }
        }),
      );
  }

  mergeImage(manifest: Manifest, layers: string[]): Observable<any> {
    return this.http.post("/dip/api/docker/merge", { manifest, layers });
  }
}
