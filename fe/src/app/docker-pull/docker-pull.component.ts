import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { DockerService, Manifest } from '../services/docker.service';
import { MessageService } from '../services/message.service';
import { FileSizePipe } from '../shared/pipes/file-size.pipe';
import { MathFloorPipe } from '../shared/pipes/math-floor.pipe';

export interface Layer {
  digest: string;
  size: number;
  downloaded: number;
  downloadProgress: number;
  uploadProgress: number;
  status: 'pending' | 'downloading' | 'uploading' | 'completed' | 'error';
}

@Component({
    selector: 'app-docker-pull',
    imports: [
        RouterModule,
        CommonModule,
        FormsModule,
        FileSizePipe,
        MathFloorPipe,
    ],
    templateUrl: './docker-pull.component.html',
    styleUrl: './docker-pull.component.css'
})
export class DockerPullComponent implements OnInit {
  imageUrl = '';
  deployment = '';
  ns = '';
  manifest: Manifest | null = null;
  layers: Layer[] = [];
  completedLayerCount: number = 0;
  isProcessing = false;
  error = '';
  configContent: any = null;
  d = '';
  constructor(
    private dockerService: DockerService,
    private messageService: MessageService,
  ) {}

  ngOnInit(): void {
    const urlParams = new URLSearchParams(window.location.search);
    this.d = urlParams.get('d') || '0';
    this.ns = urlParams.get('ns') || '';
    this.deployment = urlParams.get('dp') || '';
    this.imageUrl =
      urlParams.get('repository') ||
      'registry.cn-zhangjiakou.aliyuncs.com/ylns/nginx-empty:1.19.2';
    this.fetchManifest();
  }

  async fetchManifest() {
    try {
      this.isProcessing = true;
      this.error = '';
      this.manifest = null;
      this.configContent = null;

      // 获取manifest
      this.manifest = await firstValueFrom(
        this.dockerService.getManifest(this.imageUrl),
      );

      if (!this.manifest) {
        return;
      }
      // 解析层内容
      this.layers = this.manifest.layers.map((layer) => ({
        ...layer,
        status: 'pending',
        downloadProgress: 0,
        uploadProgress: 0,
        downloaded: 0,
      }));

      // 获取config文件内容
      this.configContent = await firstValueFrom(
        this.dockerService.downloadConfig(
          this.imageUrl,
          this.manifest.config.digest,
          this.manifest.config.size,
        ),
      );
    } catch (error: any) {
      this.error = error.message || 'Failed to fetch manifest';
      console.error(error);
    } finally {
      this.isProcessing = false;
    }
  }

  async handlePullImage() {
    if (!this.manifest) return;

    this.isProcessing = true;
    this.error = '';

    Promise.all([
      this.dockerService.uploadChunck(
        this.manifest.config.digest,
        0,
        btoa(JSON.stringify(this.configContent)),
      ),
      ...this.layers.map((layer) => this.processLayer(layer)),
    ])
      .then(() => {
        console.log('completedLayerCount:', this.completedLayerCount);

        // 确保所有任务完成后再执行合并镜像
        return firstValueFrom(
          this.dockerService.mergeImage(this.imageUrl, this.manifest),
        );
      })
      .then((r) => {
        this.messageService.publish('镜像推送成功!', true);

        if (this.d === '1') {
          // 重启服务
          return firstValueFrom(
            this.dockerService.rollout(this.ns, this.deployment),
          );
        } else {
          return new Promise(() => {});
        }
      })
      .catch((error) => {
        // 捕获任何一个任务失败的情况
        console.error('合并镜像前的任务失败：', error);
      })
      .finally(() => {
        this.isProcessing = false;
      });
  }

  private async processLayer(layer: Layer): Promise<void> {
    try {
      this.updateLayerStatus(layer.digest, 'downloading');

      await new Promise<void>((resolve, reject) => {
        this.dockerService
          .downloadLayer(this.imageUrl, layer.digest, layer.size)
          .subscribe({
            next: (progress: { p: number; u: number; d: number }) => {
              this.layers = this.layers.map((l) =>
                l.digest === layer.digest
                  ? {
                      ...l,
                      downloadProgress:
                        progress.p > 0 ? progress.p : l.downloadProgress,
                      uploadProgress:
                        progress.u > 0 ? progress.u : l.uploadProgress,
                      downloaded: progress.d > 0 ? progress.d : l.downloaded,
                    }
                  : l,
              );
            },
            error: (err) => {
              this.updateLayerStatus(layer.digest, 'error');
              reject(err);
            },
            complete: () => {
              this.updateLayerStatus(layer.digest, 'completed');
              this.completedLayerCount++;
              resolve();
            },
          });
      });
    } catch (err) {
      console.error(`Error processing layer ${layer.digest}:`, err);
      this.updateLayerStatus(layer.digest, 'error');
      throw err;
    }
  }

  private updateLayerStatus(digest: string, status: Layer['status']) {
    this.layers = this.layers.map((layer) =>
      layer.digest === digest ? { ...layer, status } : layer,
    );
  }

  getExposedPorts(): string[] {
    return this.configContent?.config?.ExposedPorts
      ? Object.keys(this.configContent.config.ExposedPorts)
      : [];
  }
}
