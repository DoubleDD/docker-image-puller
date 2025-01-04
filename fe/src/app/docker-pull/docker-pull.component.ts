import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { DockerService, Manifest } from '../services/docker.service';
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
  styleUrl: './docker-pull.component.css',
})
export class DockerPullComponent implements OnInit {
  imageUrl = '';
  newImage = '';
  oldImage = '';
  deployment = '';
  ns = '';
  manifest: Manifest | null = null;
  layers: Layer[] = [];
  completedLayerCount: number = 0;
  isProcessing = false;
  error = '';
  configContent: any = null;
  d = '';
  push = true;
  logs: string[] = [];
  constructor(private dockerService: DockerService) {}

  ngOnInit(): void {
    const urlParams = new URLSearchParams(window.location.search);
    this.d = urlParams.get('d') || '0';
    this.ns = urlParams.get('ns') || '';
    this.deployment = urlParams.get('dp') || '';
    this.push = Boolean(urlParams.get('push') || 'true');
    this.newImage = urlParams.get('newImage') || '';
    this.oldImage = urlParams.get('oldImage') || '';
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
    this.logs = [];
    this.dockerService
      .pullImage(this.newImage, this.oldImage)
      .subscribe((log) => this.addLog(log));
  }

  private addLog(log: string) {
    this.logs.push(log);
  }

  getExposedPorts(): string[] {
    return this.configContent?.config?.ExposedPorts
      ? Object.keys(this.configContent.config.ExposedPorts)
      : [];
  }
}
