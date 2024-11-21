import { CommonModule } from '@angular/common';
import {
  AfterViewInit,
  Component,
  HostListener,
  OnInit,
  ViewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink, RouterModule } from '@angular/router';
import { DownloadProgressModalComponent } from '../download-progress-modal/download-progress-modal.component';
import { stringLengthPipe } from '../shared/pipes/string-length.pipe';

export interface NsImages {
  ns: string;
  color?: string;
  deployments: Deployment[];
}
export interface Deployment {
  name: string;
  image: Image;
}
export interface Image {
  name: string;
  tag?: string;
  description: string;
}

@Component({
  selector: 'app-docker-images',
  imports: [
    RouterModule,
    CommonModule,
    FormsModule,
    stringLengthPipe,
    DownloadProgressModalComponent,
  ],
  templateUrl: './docker-images.component.html',
  styleUrl: './docker-images.component.scss',
})
export class DockerImagesComponent implements OnInit, AfterViewInit {
  @ViewChild(DownloadProgressModalComponent)
  downloadProgressModal!: DownloadProgressModalComponent;

  // 定义 Tailwind CSS 的暗色背景颜色类数组
  private colors = [
    'bg-darkGreen',
    'bg-darkOrange',
    'bg-darkYellow',
    'bg-darkBlue',
    'bg-darkIndigo',
    'bg-darkPurple',
    'bg-darkRed',
  ];
  isProcessing = false;
  error = '';
  keyword = '';
  nsImages: NsImages[] = [];
  data: NsImages[] = [
    {
      ns: 'default',
      color: this.getRandomColor(0),
      deployments: [
        {
          name: 'nginx',
          image: { name: 'nginx', description: 'nginx' },
        },
        {
          name: 'ubuntu',
          image: { name: 'ubuntu', description: 'ubuntu' },
        },
        {
          name: 'java',
          image: { name: 'openjdk', description: 'openjdk:17' },
        },
        {
          name: 'golang',
          image: { name: 'golang', description: 'golang:2.4', tag: '2.4' },
        },
        {
          name: 'node',
          image: { name: 'node', description: 'node:22.1', tag: '22.2' },
        },
      ],
    },
    {
      ns: 'ylns',
      color: this.getRandomColor(1),
      deployments: [
        {
          name: 'busybox',
          image: {
            name: 'busybox',
            tag: '1.19.2',
            description:
              '172.27.35.4:5000/yunli_mid_platform/busybox:latest-arm64',
          },
        },
      ],
    },
  ];
  registry = 'registry.cn-zhangjiakou.aliyuncs.com';
  namespace = '';
  d = '';

  constructor(private router: Router) {}

  ngOnInit(): void {
    const urlParams = new URLSearchParams(window.location.search);
    this.namespace = urlParams.get('namespace') || '';
    this.d = urlParams.get('d') || '';
    const debug = urlParams.get('debug') || '';
    if (debug) {
      this.nsImages = [...this.data];
      return;
    }
    // 获取镜像列表
    fetch(`/dip/api/docker/images?namespace=${this.namespace}`).then(
      async (resp) => {
        const json = await resp.json();
        const arr: NsImages[] = [];
        let i = 0;
        for (const key of json) {
          const dps: Deployment[] = [];
          (key.Deployments as any[]).forEach((deployment) => {
            const imageUrl = deployment.ImageName;
            const imageArr = imageUrl.split('\n');
            for (const image of imageArr) {
              const item = parseDockerImage(image);
              const imageName = item.image;
              const tag = item.tag || '';
              const dp = {
                name: deployment.Name,
                image: {
                  name: imageName,
                  description: image,
                  tag: tag,
                },
              };
              dps.push(dp);
            }
          });

          arr.push({
            ns: key.Namespace,
            color: this.getRandomColor(i++),
            deployments: dps.sort((a, b) => a.name.localeCompare(b.name)),
          });
        }
        this.data = [...arr];
        this.nsImages = [...this.data];
      },
    );
  }

  ngAfterViewInit(): void {}

  @HostListener('document:keydown.control.k', ['$event'])
  onKeydownHandler(event: KeyboardEvent) {
    event.preventDefault(); // 阻止默认行为
    this.focusSearchInput();
  }

  // 生成随机颜色的方法
  getRandomColor(index?: number) {
    const random = Math.floor(Math.random() * 100) + 1;
    return this.colors[(index || random) % this.colors.length];
  }

  focusSearchInput() {
    const inputElement = document.querySelector(
      'input[type="text"]',
    ) as HTMLInputElement;
    if (inputElement) {
      inputElement.focus();
    }
  }

  search() {
    const result: NsImages[] = [];
    this.data.forEach((v) => {
      const deployments = v.deployments.filter(
        (e) => e.image.description.indexOf(this.keyword) > -1,
      );
      if (deployments.length > 0) {
        result.push({ ns: v.ns, color: v.color, deployments });
      }
    });

    this.nsImages = result;
  }

  goToDetail(ns: string, deploymentName: string, imageName: string) {
    this.router.navigate(['/detail'], {
      queryParams: {
        d: this.d,
        ns: ns,
        dp: deploymentName,
        repository: imageName.replace(
          '172.27.35.4:5000',
          'registry.cn-zhangjiakou.aliyuncs.com',
        ),
      },
    });
  }
  goToLog(ns: string, deploymentName: string) {
    window.open(`./pods?ns=${ns}&dp=${deploymentName}`, '_blank');
  }

  pull(event: Event, imageName: string) {
    event.stopPropagation();

    // 模拟镜像分层数据
    const layers = [
      { size: 100, progress: 0, uProgress: 0 },
      { size: 200, progress: 0, uProgress: 0 },
      { size: 300, progress: 0, uProgress: 0 },
      { size: 400, progress: 0, uProgress: 0 },
      { size: 500, progress: 0, uProgress: 0 },
      { size: 600, progress: 0, uProgress: 0 },
    ];

    const totalSize = layers.reduce((sum, layer) => sum + layer.size, 0);

    // 显示弹窗
    this.downloadProgressModal.layers = layers;
    this.downloadProgressModal.totalSize = totalSize;
    this.downloadProgressModal.imageName = imageName;
    this.downloadProgressModal.show();

    // 模拟下载进度更新
    layers.forEach((layer, index) => {
      const interval = setInterval(() => {
        if (layer.progress < 100) {
          layer.progress += 20;
          this.downloadProgressModal.updateLayerProgress(index, layer.progress);
        }
        if (layer.uProgress < 100) {
          layer.uProgress += 10;
          this.downloadProgressModal.updateLayerUploadProgress(
            index,
            layer.uProgress,
          );
        }

        if (layer.uProgress >= 100 && layer.progress >= 100) {
          clearInterval(interval);
        }
      }, 500);
    });
  }
}

interface DockerImage {
  registry?: string;
  repository: string;
  image: string;
  tag?: string;
  digest?: string;
}

function parseDockerImage(imageUrl: string): DockerImage {
  const result: DockerImage = {
    repository: '',
    image: '',
  };

  // 匹配 Docker 镜像地址的正则表达式
  const dockerImageRegex =
    /^(?:(?<registry>[^/]+)\/)?(?:(?<repository>[^/:]+)\/)?(?<image>[^:@]+)(?::(?<tag>[^:@]+))?(?:@(?<digest>[^:@]+))?$/;
  const match = imageUrl.match(dockerImageRegex);

  if (match) {
    result.registry = match.groups?.['registry'];
    result.repository = match.groups?.['repository'] || '';
    result.image = match.groups?.['image'] || '';
    result.tag = match.groups?.['tag'];
    result.digest = match.groups?.['digest'];
  } else {
    throw new Error('Invalid Docker image format');
  }

  return result;
}
