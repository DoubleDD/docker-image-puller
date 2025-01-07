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
  splitPrefix: string[] = [];

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

  parentFiles(bucket: string, part: string, index: number) {
    this.currentPrefix = this.splitPrefix.slice(0, index + 1).join('/') + '/';
    console.log(this.currentPrefix);
    this.loadFiles(bucket, this.currentPrefix);
  }

  // 加载文件列表
  loadFiles(bucket: string, prefix: string = ''): void {
    this.currentBucket = bucket;
    this.currentPrefix = prefix;
    this.splitPrefix = this.currentPrefix
      .split('/')
      .filter((part) => part.length > 0);
    this.minioService.listFiles(bucket, prefix).subscribe(
      (response: any) => {
        this.files = response;
      },
      (error) => {
        console.error('Error loading files:', error);
      },
    );
  }

  // 获取文件元数据
  getMetadata(bucket: string, objectName: string): void {
    if (objectName.endsWith('/')) {
      // 获取下一级文件
      this.loadFiles(bucket, objectName);
    } else {
      this.minioService.getFileMetadata(bucket, objectName).subscribe({
        next: (response: any) => {
          this.metadata = response;
        },
        error: (err) => {
          console.error('Error fetching metadata:', err);
        },
      });
    }
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
