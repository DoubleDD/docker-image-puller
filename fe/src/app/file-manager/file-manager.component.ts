import { Component } from '@angular/core';
import { MinioService } from '../services/minio.service';
import { CommonModule, JsonPipe } from '@angular/common';

@Component({
  selector: 'app-file-manager',
  imports: [CommonModule, JsonPipe],
  templateUrl: './file-manager.component.html',
  styleUrl: './file-manager.component.scss',
})
export class FileManagerComponent {
  buckets: string[] = [];
  files: string[] = [];
  currentBucket: string = '';
  currentPrefix: string = '';
  metadata: any = {};

  constructor(private minioService: MinioService) {}

  ngOnInit(): void {
    this.loadBuckets();
  }

  // 加载 Bucket 列表
  loadBuckets(): void {
    this.minioService.listBuckets().subscribe(
      (response: any) => {
        this.buckets = response.map((bucket: any) => bucket.name);
      },
      (error) => {
        console.error('Error loading buckets:', error);
      },
    );
  }

  // 加载文件列表
  loadFiles(bucket: string, prefix: string = ''): void {
    this.currentBucket = bucket;
    this.currentPrefix = prefix;
    this.minioService.listFiles(bucket, prefix).subscribe(
      (response: any) => {
        this.files = response.contents.map((item: any) => item.key);
      },
      (error) => {
        console.error('Error loading files:', error);
      },
    );
  }

  // 获取文件元数据
  getMetadata(bucket: string, objectName: string): void {
    this.minioService.getFileMetadata(bucket, objectName).subscribe(
      (response: any) => {
        this.metadata = response.headers;
      },
      (error) => {
        console.error('Error fetching metadata:', error);
      },
    );
  }

  // 上传文件
  onFileSelected(event: any): void {
    const file: File = event.target.files[0];
    if (file) {
      const objectName = `${this.currentPrefix}${file.name}`;
      this.minioService
        .uploadFile(this.currentBucket, objectName, file)
        .subscribe(
          () => {
            this.loadFiles(this.currentBucket, this.currentPrefix);
          },
          (error) => {
            console.error('Error uploading file:', error);
          },
        );
    }
  }
}
