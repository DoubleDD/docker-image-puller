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

  constructor(private http: HttpClient) { }

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

  downloadLayer(imageUrl: string, digest: string, size: number): Observable<{ p: number, u: number, d: number }> {
    return new Observable<{ p: number, u: number, d: number }>(observer => {
      let totalUploaded = 0;

      // 创建SSE连接
      const eventSource = new EventSource(`/dip/api/docker/blob/download?image=${imageUrl}&digest=${digest}&size=${size}`, {
        withCredentials: true
      });

      // 监听任务ID
      eventSource.addEventListener('taskId', (event) => {
        console.log('Download task created:', event.data);
      });

      // 监听任务数据
      eventSource.addEventListener('data', async (event) => {
        try {
          // 创建 FormData 对象
          const formData = new FormData();
          formData.append('digest', digest);
          formData.append('chunk', event.data); // event.data 已经是 base64 编码的数据
          // 计算当前块的大小（解码base64后的大小）
          const chunkSize = Math.ceil(event.data.length * 3 / 4); // 估算base64解码后的大小

          console.log("上传层分块，分块大小", chunkSize);

          // 上传数据块
          await this.http.post('/dip/api/docker/blob/chunk', formData).toPromise();

          // 更新已上传的总大小
          totalUploaded += chunkSize;

          // 计算上传进度百分比
          const uploadPercentage = (totalUploaded / size) * 100;

          // 发送最新的下载和上传进度
          observer.next({ p: -1, u: uploadPercentage, d: -1 });

        } catch (error) {
          console.error('Failed to upload chunk:', error);
          observer.error('上传数据块失败');
          eventSource.close();
        }
      });

      // 监听进度更新
      eventSource.addEventListener('progress', (event) => {
        const progress = JSON.parse(event.data);
        if (progress.digest === digest) {
          observer.next({ p: progress.percentage, u: -1, d: progress.downloaded });
        }
      });

      // 监听完成事件
      eventSource.addEventListener('complete', (event) => {
        const result = JSON.parse(event.data);
        if (result.digest === digest) {
          observer.next({ p: 100, u: 100, d: result.size });
          observer.complete();
          eventSource.close();
        }
      });

      // 监听错误
      eventSource.addEventListener('error', (event: any) => {
        const errorMessage = event.data || '下载失败';
        observer.error(errorMessage);
        eventSource.close();
      });

      // 清理函数
      return () => {
        eventSource.close();
      };
    });
  }



  mergeImage(image: string, manifest: Manifest, layers: string[]): Observable<any> {
    return this.http.post("/dip/api/docker/merge", { image, manifest, layers });
  }

  // 添加配置文件下载方法
  downloadConfig(imageUrl: string, digest: string, size: number): Observable<any> {
    return new Observable(observer => {
      const eventSource = new EventSource(
        `/dip/api/docker/blob/download?image=${imageUrl}&digest=${digest}&size=${size}`,
        { withCredentials: true }
      );

      let configData = '';

      eventSource.addEventListener('data', async (event) => {
        configData += event.data;
        try {
          // 创建 FormData 对象
          const formData = new FormData();
          formData.append('digest', digest);
          formData.append('chunk', event.data); // event.data 已经是 base64 编码的数据

          // 上传数据块
          await this.http.post('/dip/api/docker/blob/chunk', formData).toPromise();

        } catch (error) {
          console.error('Failed to upload chunk:', error);
          observer.error('上传config数据块失败');
          eventSource.close();
        }
      });

      eventSource.addEventListener('complete', () => {
        try {
          // base64解码并解析JSON
          const decodedData = atob(configData);
          const config = JSON.parse(decodedData);
          observer.next(config);
          observer.complete();
        } catch (error) {
          observer.error('Failed to parse config data');
        }
        eventSource.close();
      });

      eventSource.addEventListener('error', (event: any) => {
        observer.error(event.data || 'Failed to download config');
        eventSource.close();
      });

      return () => {
        eventSource.close();
      };
    });
  }
}
