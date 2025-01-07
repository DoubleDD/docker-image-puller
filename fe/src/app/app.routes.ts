import { Routes } from '@angular/router';
import { K8sLogsComponent } from './k8s-logs/k8s-logs.component';

export const routes: Routes = [
  {
    path: 'file-manager',
    title: 'File Manager',
    loadComponent: () =>
      import('./file-manager/file-manager.component').then(
        (m) => m.FileManagerComponent,
      ),
  },
  {
    path: 'http-client',
    title: 'Http Client',
    loadComponent: () =>
      import('./http-client/http-client.component').then(
        (m) => m.HttpClientComponent,
      ),
  },
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
    path: 'docker-add',
    title: 'Add Docker Images',
    loadComponent: () =>
      import('./add-image/add-image.component').then(
        (m) => m.AddImageComponent,
      ),
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
