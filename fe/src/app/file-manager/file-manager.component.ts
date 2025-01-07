import { Component, ViewChild } from '@angular/core';
import { MinioService } from '../services/minio.service';
import { CommonModule, JsonPipe } from '@angular/common';
import { forkJoin } from 'rxjs';

@Component({
  selector: 'app-file-manager',
  imports: [CommonModule, JsonPipe],
  templateUrl: './file-manager.component.html',
  styleUrl: './file-manager.component.scss',
})
export class FileManagerComponent {
  @ViewChild('fileInput') fileInput: any;

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

  check(event: any) {
    if (!this.currentBucket) {
      alert('请先选择Bucket');
      event.preventDefault();
      return;
    }
  }
  // 上传文件
  onFileSelected(event: any): void {
    const files: File[] = event.target.files;
    if (files.length > 0) {
      // 创建一个数组来存储所有上传的 Observable
      const uploadObservables = [];

      for (let i = 0; i < files.length; i++) {
        const file = files[i];

        const objectName = `${this.currentPrefix}${file.webkitRelativePath || file.name}`;

        // 将每个文件的上传操作转换为 Observable 并存入数组
        const uploadObservable = this.minioService.uploadFile(
          this.currentBucket,
          objectName,
          file,
        );
        uploadObservables.push(uploadObservable);
      }

      // 使用 forkJoin 等待所有上传操作完成
      forkJoin(uploadObservables).subscribe({
        next: () => {
          // 所有文件上传完成后，刷新文件列表
          this.loadFiles(this.currentBucket, this.currentPrefix);
          // 清除 input 的值
          this.clearFileInput();
        },
        error: (error) => {
          console.error('Error uploading files:', error);
          // 即使有错误，也清除 input 的值
          this.clearFileInput();
        },
      });
    }
  }

  // 清空 input 的值
  clearFileInput(): void {
    if (this.fileInput) {
      this.fileInput.nativeElement.value = ''; // 清空 input 的值
    }
  }
}
