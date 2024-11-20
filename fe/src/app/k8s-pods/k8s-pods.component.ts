import { Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { K8sService } from '../services/k8s.service';
import { Subscription } from 'rxjs';
import { K8sLogsComponent } from '../k8s-logs/k8s-logs.component';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-k8s-pods',
  standalone: true,
  imports: [CommonModule, RouterModule, K8sLogsComponent],
  templateUrl: './k8s-pods.component.html',
  styleUrl: './k8s-pods.component.scss',
})
export class K8sPodsComponent implements OnInit, OnDestroy {
  pods: string[] = [];
  private subscription!: Subscription;
  ns: string = '';
  deployment: string = '';
  constructor(
    private route: ActivatedRoute,
    private k8s: K8sService,
  ) {}

  ngOnInit(): void {
    // 从查询参数中获取其他参数
    this.route.queryParams.subscribe((params) => {
      this.ns = params['ns'];
      this.deployment = params['dp'];
    });

    this.subscription = this.k8s
      .getPods(this.ns, this.deployment)
      .subscribe((resp: any) => {
        this.pods = resp.list;
      });
  }
  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }
}
