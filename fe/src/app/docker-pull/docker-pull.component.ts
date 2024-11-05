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

  constructor(
    private dockerService: DockerService,
    private http: HttpClient,
  ) {}

  ngOnInit() {}

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
    if (!this.imageUrl) return;

    try {
      this.isProcessing = true;
      this.error = "";
      const parsedUrl = this.parseImageUrl(this.imageUrl);

      this.manifest = await firstValueFrom(
        this.dockerService.getManifest(parsedUrl),
      );

      if (this.manifest) {
        this.layers = this.manifest.layers.map((layer) => ({
          digest: layer.digest,
          size: layer.size,
          downloadProgress: 0,
          uploadProgress: 0,
          status: "pending",
        }));
      }
    } catch (err) {
      this.error = "Failed to fetch manifest";
      console.error(err);
    } finally {
      this.isProcessing = false;
    }
  }

  private async processLayer(layer: Layer): Promise<void> {
    try {
      // Update status to downloading
      this.updateLayerStatus(layer.digest, "downloading");

      // Download layer
      const blob = await this.dockerService
        .downloadLayer(this.imageUrl,layer.digest,layer.size)
        .pipe(
          finalize(() => {
            if (layer.downloadProgress < 100) {
              this.updateLayerStatus(layer.digest, "error");
            }
          }),
        )
        .toPromise();

      if (!blob) throw new Error("Failed to download layer");

      // Update status to uploading
      this.updateLayerStatus(layer.digest, "uploading");

      // Upload layer
      await this.dockerService
        .uploadLayer(blob, layer.digest)
        .pipe(
          finalize(() => {
            if (layer.uploadProgress < 100) {
              this.updateLayerStatus(layer.digest, "error");
            }
          }),
        )
        .toPromise();

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
}
