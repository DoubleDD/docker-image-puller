import { CommonModule } from "@angular/common";
import { Component } from "@angular/core";
import { FormsModule } from "@angular/forms";
import { firstValueFrom } from "rxjs";
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
export class DockerPullComponent {
  imageUrl = "registry.cn-zhangjiakou.aliyuncs.com/yunli_mid_platform/resource:dtwin-etl-shg-be-1.0.0-xc-arm64";
  manifest: Manifest | null = null;
  layers: Layer[] = [];
  isProcessing = false;
  error = "";
  configContent: any = null;

  constructor(private dockerService: DockerService) {}

  async fetchManifest() {
    try {
      this.isProcessing = true;
      this.error = '';
      this.manifest = null;
      this.configContent = null;

      // 获取manifest
      this.manifest = await firstValueFrom(this.dockerService.getManifest(this.imageUrl));

      // 解析层内容
      this.layers = this.manifest.layers.map(layer => ({
        ...layer,
        status: 'pending',
        downloadProgress: 0,
        uploadProgress: 0,
        downloaded: 0
      }));

      // 获取config文件内容
      this.configContent = await firstValueFrom(this.dockerService.downloadConfig(
        this.imageUrl,
        this.manifest.config.digest,
        this.manifest.config.size
      ));

      // 下载镜像各个层的内容
      // this.handlePullImage()

    } catch (error: any) {
      this.error = error.message || 'Failed to fetch manifest';
    } finally {
      this.isProcessing = false;
    }
  }

  private async processLayer(layer: Layer): Promise<void> {
    try {
      this.updateLayerStatus(layer.digest, "downloading");

      await new Promise<void>((resolve, reject) => {
        this.dockerService.downloadLayer(this.imageUrl, layer.digest, layer.size)
          .subscribe({
            next: (progress: { p: number, u: number, d: number }) => {
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
      await Promise.all(this.layers.map((layer) => this.processLayer(layer)));
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

  getExposedPorts(): string[] {
    return this.configContent?.config?.ExposedPorts
      ? Object.keys(this.configContent.config.ExposedPorts)
      : [];
  }
}
