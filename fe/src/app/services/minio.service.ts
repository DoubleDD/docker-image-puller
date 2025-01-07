import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

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
    return this.http.put(
      `/dip/minio/upload?bucket=${bucketName}&object=${objectName}`,
      file,
    );
  }
}
