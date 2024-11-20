import { Routes } from '@angular/router';
import { K8sLogsComponent } from './k8s-logs/k8s-logs.component';

export const routes: Routes = [
  {
    path: 'pods',
    title: 'Pod List',
    loadComponent: () =>
      import('./k8s-pods/k8s-pods.component').then((m) => m.K8sPodsComponent),
  },
  {
    path: 'logs',
    title: 'Pod Logs',
    component: K8sLogsComponent,
  },
  {
    path: 'docker',
    title: 'Docker Images',
    loadComponent: () =>
      import('./docker-images/docker-images.component').then(
        (m) => m.DockerImagesComponent,
      ),
  },
  {
    path: 'detail',
    title: 'Docker Pull',
    loadComponent: () =>
      import('./docker-pull/docker-pull.component').then(
        (m) => m.DockerPullComponent,
      ),
  },
  { path: '', redirectTo: '/docker', pathMatch: 'full' }, // 默认重定向到 '/docker'
  { path: '**', redirectTo: '/docker' }, // 未匹配到的路径自动重定向到 '/docker'
];
