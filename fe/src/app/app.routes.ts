import { Routes } from '@angular/router';
import { DockerPullComponent } from './docker-pull/docker-pull.component';
import { DockerImagesComponent } from './docker-images/docker-images.component';

export const routes: Routes = [
  {
    path: 'docker',
    title: 'Docker Images',
    component: DockerImagesComponent,
  },
  {
    path: 'detail',
    title: 'Docker Pull',
    component: DockerPullComponent,
  },
  { path: '', redirectTo: '/docker', pathMatch: 'full' }, // 默认重定向到 '/docker'
  { path: '**', redirectTo: '/docker' }, // 未匹配到的路径自动重定向到 '/docker'
];
