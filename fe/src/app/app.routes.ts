import { Routes } from '@angular/router';
import { DockerPullComponent } from './docker-pull/docker-pull.component';

export const routes: Routes = [
  {
    path: "docker",
    title: "Docker Pull",
    component: DockerPullComponent,
  },
];
