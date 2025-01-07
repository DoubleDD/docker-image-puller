import {
  HttpClient,
  HttpEvent,
  HttpEventType,
  HttpHeaders,
  HttpRequest,
} from '@angular/common/http';
import { Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class MinioService {
  constructor(private http: HttpClient) {}

  // 获取 Bucket 列表
  listBuckets(): Observable<any> {
    return this.http.get(`/dip/minio/buckets`);
  }

  // 列出指定目录下的文件
  listFiles(bucketName: string, prefix: string): Observable<any> {
    return this.http.get(
      `/dip/minio/files?bucket=${bucketName}&prefix=${prefix}`,
    );
  }

  // 获取文件元数据
  getFileMetadata(bucketName: string, objectName: string): Observable<any> {
    return this.http.get(
      `/dip/minio/metadata?bucket=${bucketName}&object=${objectName}`,
    );
  }

  // 上传文件
  uploadFile(
    bucketName: string,
    objectName: string,
    file: File,
  ): Observable<any> {
    // 创建 FormData 对象
    const formData = new FormData();
    formData.append('file', file, file.name);

    // 设置请求头
    const headers = new HttpHeaders({
      Accept: 'application/json',
    });

    // 创建带有进度监听的请求
    const req = new HttpRequest(
      'PUT',
      `/dip/minio/upload?bucket=${bucketName}&object=${objectName}`,
      formData,
      {
        headers: headers,
        reportProgress: true, // 启用进度监听
      },
    );

    return this.http.request(req).pipe(
      map((event: HttpEvent<any>) => {
        switch (event.type) {
          case HttpEventType.UploadProgress:
            // 计算上传进度
            const progress = Math.round(
              (100 * event.loaded) / (event.total || 1),
            );
            return { progress };

          case HttpEventType.Response:
            // 上传完成，返回服务器响应
            return { progress: 100, response: event.body };

          default:
            return { progress: 0 };
        }
      }),
    );
  }
}
