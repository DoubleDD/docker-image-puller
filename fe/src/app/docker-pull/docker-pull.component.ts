import { CommonModule } from "@angular/common";
import { HttpClient } from "@angular/common/http";
import { Component, OnInit } from "@angular/core";
import { FormsModule } from "@angular/forms";
import { finalize, firstValueFrom } from "rxjs";
import { DockerService } from "./docker.service";
import { FileSizePipe } from '../shared/pipes/file-size.pipe';

export interface Layer {
  digest: string;
  size: number;
  downloaded: number;
  downloadProgress: number;
  uploadProgress: number;
  status: "pending" | "downloading" | "uploading" | "completed" | "error";
}

export interface Manifest {
  schemaVersion: number;
  mediaType: string;
  config: {
    digest: string;
    size: number;
  };
  layers: {
    mediaType: string;
    size: number;
    digest: string;
  }[];
}

@Component({
  selector: "app-docker-pull",
  standalone: true,
  imports: [CommonModule, FormsModule, FileSizePipe],
  templateUrl: "./docker-pull.component.html",
  styleUrl: "./docker-pull.component.css",
})
export class DockerPullComponent implements OnInit {
  imageUrl =
    "registry.cn-zhangjiakou.aliyuncs.com/yunli_mid_platform/resource:dtwin-etl-shg-be-1.0.0-xc-arm64";
  manifest: Manifest | null = null;
  layers: Layer[] = [];
  isProcessing = false;
  error = "";
  configContent: any = null;

  constructor(
    private dockerService: DockerService,
    private http: HttpClient,
  ) { }

  ngOnInit() { }

  parseImageUrl(url: string): string {
    if (!url.includes("/")) {
      return `registry.hub.docker.com/library/${url}`;
    }
    return url;
  }

  updateLayerProgress(
    digest: string,
    type: "download" | "upload",
    progress: number,
  ) {
    this.layers = this.layers.map((layer) =>
      layer.digest === digest
        ? {
          ...layer,
          [type === "download" ? "downloadProgress" : "uploadProgress"]:
            progress,
        }
        : layer,
    );
  }

  async fetchManifest() {
    try {
      this.isProcessing = true;
      this.error = '';
      this.manifest = null;
      this.configContent = null;

      // 获取manifest
      this.manifest = await firstValueFrom(this.dockerService.getManifest(this.imageUrl));

      // 下载配置文件
      this.configContent = await firstValueFrom(this.dockerService.downloadConfig(
        this.imageUrl,
        this.manifest.config.digest,
        this.manifest.config.size
      ));

      // 处理层信息
      this.layers = this.manifest.layers.map(layer => ({
        ...layer,
        status: 'pending',
        downloadProgress: 0,
        uploadProgress: 0,
        downloaded: 0
      }));
    } catch (error: any) {
      this.error = error.message || 'Failed to fetch manifest';
    } finally {
      this.isProcessing = false;
    }
  }

  private async processLayer(layer: Layer): Promise<void> {
    try {
      // Update status to downloading
      this.updateLayerStatus(layer.digest, "downloading");

      // Download layer and handle progress
      await new Promise<void>((resolve, reject) => {
        this.dockerService.downloadLayer(this.imageUrl, layer.digest, layer.size)
          .subscribe({
            next: (progress: { p: number, u: number, d: number }) => {
              // 更新下载进度
              this.layers = this.layers.map(l =>
                l.digest === layer.digest
                  ? {
                    ...l,
                    downloadProgress: progress.p > 0 ? progress.p : l.downloadProgress,
                    uploadProgress: progress.u > 0 ? progress.u : l.uploadProgress,
                    downloaded: progress.d > 0 ? progress.d : l.downloaded
                  }
                  : l
              );
            },
            error: (err) => {
              this.updateLayerStatus(layer.digest, "error");
              reject(err);
            },
            complete: () => {
              this.updateLayerStatus(layer.digest, "completed");
              resolve();
            }
          });
      });

      // Update status to uploading
      this.updateLayerStatus(layer.digest, "uploading");

      // Update status to completed
      this.updateLayerStatus(layer.digest, "completed");
    } catch (err) {
      console.error(`Error processing layer ${layer.digest}:`, err);
      this.updateLayerStatus(layer.digest, "error");
      throw err;
    }
  }

  private updateLayerStatus(digest: string, status: Layer["status"]) {
    this.layers = this.layers.map((layer) =>
      layer.digest === digest ? { ...layer, status } : layer,
    );
  }

  async handlePullImage() {
    if (!this.manifest) return;

    this.isProcessing = true;
    this.error = "";

    try {
      // Process all layers in parallel
      await Promise.all(this.layers.map((layer) => this.processLayer(layer)));

      // Notify server to merge layers
      await this.dockerService
        .mergeImage(
          this.imageUrl,
          this.manifest,
          this.layers.map((l) => l.digest),
        )
        .toPromise();
    } catch (err) {
      this.error = "Failed to process image layers";
      console.error(err);
    } finally {
      this.isProcessing = false;
    }
  }

  // 辅助方法：获取暴露的端口列表
  getExposedPorts(): string[] {
    if (!this.configContent?.config?.ExposedPorts) {
      return [];
    }
    return Object.keys(this.configContent.config.ExposedPorts);
  }
}
