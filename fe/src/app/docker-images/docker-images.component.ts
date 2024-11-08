import { CommonModule } from '@angular/common';
import { Component, OnInit, HostListener } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { stringLengthPipe } from '../shared/pipes/string-length.pipe';
import { Router } from '@angular/router';

export interface NsImages {
  ns: string;
  color?: string;
  images: Image[];
}
export interface Image {
  name: string;
  tag?: string;
  description: string;
}
@Component({
  selector: 'app-docker-images',
  standalone: true,
  imports: [CommonModule, FormsModule, stringLengthPipe],
  templateUrl: './docker-images.component.html',
  styleUrl: './docker-images.component.scss',
})
export class DockerImagesComponent implements OnInit {
  isProcessing = false;
  error = '';
  keyword = '';
  data: NsImages[] = [];
  nsImages: NsImages[] = [
    {
      ns: 'default',
      images: [
        { name: 'nginx', description: 'nginx' },
        { name: 'ubuntu', description: 'ubuntu' },
        { name: 'openjdk', description: 'java17' },
        { name: 'golang', description: 'golang' },
      ],
    },
    {
      ns: 'ylns',
      images: [
        {
          name: 'ylns/nginx-empty',
          tag: '1.19.2',
          description: 'nginx-empty',
        },
      ],
    },
  ];
  registry = 'registry.cn-zhangjiakou.aliyuncs.com';
  namespace = '';
  constructor(private router: Router) {}
  ngOnInit(): void {
    const urlParams = new URLSearchParams(window.location.search);
    this.namespace = urlParams.get('namespace') || '';
    // 获取镜像列表
    fetch(`/dip/api/docker/images?namespace=${this.namespace}`).then(
      async (resp) => {
        const json = await resp.json();
        const arr: NsImages[] = [];
        for (const key in json) {
          arr.push({
            ns: key,
            color: this.getRandomColor(),
            images: (json[key] as string[]).map((e) => {
              const iarr = e.split(';');
              return { name: iarr[0], description: e, tag: iarr[1] };
            }),
          });
        }
        this.data = [...arr];
        this.nsImages = [...this.data];
      },
    );
  }

  @HostListener('document:keydown.control.k', ['$event'])
  onKeydownHandler(event: KeyboardEvent) {
    event.preventDefault(); // 阻止默认行为
    this.focusSearchInput();
  }

  // 生成随机颜色的方法
  getRandomColor() {
    const letters = '0123456789ABCDEF';
    let color = '#';
    for (let i = 0; i < 6; i++) {
      color += letters[Math.floor(Math.random() * 16)];
    }
    return color;
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
      const images = v.images.filter(
        (e) => e.description.indexOf(this.keyword) > -1,
      );
      result.push({ ns: v.ns, images });
    });

    this.nsImages = result;
  }

  goToDetail(imageName: string) {
    this.router.navigate(['/detail'], {
      queryParams: { repository: imageName },
    });
  }
}
