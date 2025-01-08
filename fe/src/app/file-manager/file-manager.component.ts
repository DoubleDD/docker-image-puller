import { Component, ElementRef, HostListener, ViewChild } from '@angular/core';
import { MinioService } from '../services/minio.service';
import { CommonModule, JsonPipe } from '@angular/common';
import { forkJoin } from 'rxjs';
import { FileSizePipe } from '../shared/pipes/file-size.pipe';

@Component({
  selector: 'app-file-manager',
  imports: [CommonModule, JsonPipe, FileSizePipe],
  templateUrl: './file-manager.component.html',
  styleUrl: './file-manager.component.scss',
})
export class FileManagerComponent {
  @ViewChild('fileInput') fileInput: any;
  @ViewChild('dropArea') dropArea!: ElementRef;

  uploading = false;
  uploadSuccess = false;
  uploadError: string | null = null;

  buckets: string[] = [];
  files: any[] = [];
  currentBucket: string = '';
  currentPrefix: string = '';
  metadata: any = null;
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
        this.files = this.sortFiles(response);
        this.metadata = null;
      },
      (error) => {
        console.error('Error loading files:', error);
      },
    );
  }
  /**
   * 对文件列表进行排序
   * 1. 文件夹排在前面，文件排在后面
   * 2. 每个类型中按字母顺序排序
   */
  sortFiles(files: any[]): any[] {
    return files.sort((a, b) => {
      const isFolderA = a.name.endsWith('/');
      const isFolderB = b.name.endsWith('/');

      // 如果 a 是文件夹而 b 是文件，a 排在前面
      if (isFolderA && !isFolderB) {
        return -1;
      }
      // 如果 a 是文件而 b 是文件夹，b 排在前面
      if (!isFolderA && isFolderB) {
        return 1;
      }
      // 如果都是文件夹或都是文件，按字母顺序排序
      return a.name.localeCompare(b.name);
    });
  }

  // 获取文件元数据
  getMetadata(bucket: string, object: any): void {
    console.log(object);
    const objectName = object.name;

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
    this.fileInfos = event.target.files.map((file: File) => {
      return {
        path: file.webkitRelativePath || file.name,
        file: file,
      };
    });
    this.uploadFiles(this.fileInfos);
  }

  uploadFiles(files: { path: string; file: File }[]): void {
    if (files.length > 0) {
      // 创建一个数组来存储所有上传的 Observable
      const uploadObservables = [];

      for (let i = 0; i < files.length; i++) {
        const file = files[i];

        const objectName = `${this.currentPrefix}${file.path}`;

        // 将每个文件的上传操作转换为 Observable 并存入数组
        const uploadObservable = this.minioService.uploadFile(
          this.currentBucket,
          objectName,
          file.file,
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

  // 拖放
  onDragOver(event: DragEvent) {
    event.preventDefault(); // 阻止默认行为
    this.dropArea.nativeElement.classList.add('active');
  }

  onDragEnter(event: DragEvent) {
    event.preventDefault();
    this.dropArea.nativeElement.classList.add('active');
  }

  onDragLeave(event: DragEvent) {
    event.preventDefault();
    this.dropArea.nativeElement.classList.remove('active');
  }

  async onDrop(event: DragEvent) {
    event.preventDefault();
    this.dropArea.nativeElement.classList.remove('active');

    this.uploadSuccess = false;
    this.uploadError = null;
    this.uploading = false;
    this.fileInfos = [];

    const items = event.dataTransfer?.items || [];

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.kind === 'file') {
        const entry = item.webkitGetAsEntry();
        if (entry) {
          await this.handleEntry(entry, '');
        }
      }
    }

    this.uploadFiles(this.fileInfos);
  }

  fileInfos: { path: string; file: File }[] = [];

  async handleEntry(entry: FileSystemEntry, path: string): Promise<void> {
    if (entry.isFile) {
      try {
        const file = await new Promise<File>((resolve, reject) => {
          (entry as FileSystemFileEntry).file(resolve, reject);
        });
        this.fileInfos.push({ path: path + entry.name, file: file });
        console.log('文件信息:', { path: path + entry.name, file: file });
      } catch (error) {
        console.error('读取文件出错:', error);
      }
    } else if (entry.isDirectory) {
      console.log('文件夹名:', entry.name);
      const reader = (entry as FileSystemDirectoryEntry).createReader();
      await this.readEntries(reader, path + entry.name + '/'); // 路径添加文件夹名
    }
  }

  async readEntries(
    reader: FileSystemDirectoryReader,
    path: string,
  ): Promise<void> {
    try {
      const entries = await new Promise<FileSystemEntry[]>(
        (resolve, reject) => {
          reader.readEntries(resolve, reject);
        },
      );

      if (entries.length) {
        for (let i = 0; i < entries.length; i++) {
          await this.handleEntry(entries[i], path); // 递归调用，并传递当前路径
        }
        // 继续读取，直到 entries 为空
        await this.readEntries(reader, path);
      }
    } catch (error) {
      console.error('读取文件夹内容出错:', error);
    }
  }
}
