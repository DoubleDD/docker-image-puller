import {
  Component,
  ElementRef,
  Input,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';
import { K8sService } from '../services/k8s.service';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-k8s-logs',
  standalone: true,
  imports: [FormsModule, CommonModule],
  templateUrl: './k8s-logs.component.html',
  styleUrl: './k8s-logs.component.scss',
})
export class K8sLogsComponent implements OnInit, OnDestroy {
  @ViewChild('logBody', { static: false }) logBody!: ElementRef;
  logs: string[] = [];
  autoScroll: boolean = false;
  startTime: string = '';
  endTime: string = '';
  private subscription!: Subscription;

  @Input() ns: string = '';
  @Input() podName: string = '';

  constructor(
    private logsService: K8sService,
    private route: ActivatedRoute,
  ) {}

  ngOnInit(): void {
    // 从查询参数中获取其他参数
    this.route.queryParams.subscribe((params) => {
      const ns = params['ns'];
      const podName = params['p'];
      if (ns) {
        this.ns = ns;
      }
      if (podName) {
        this.podName = podName;
      }
      if (!this.podName) {
        this.podName = 'chat2db-c49957b9-jwt9p';
      }
    });

    this.subscription = this.logsService
      .getLogs(this.ns, this.podName, '')
      .subscribe((log) => {
        if (log.trim() != '') {
          log
            .trimEnd()
            .split('\n')
            .forEach((l) => this.logs.push(l));
          this.scrollToBottom();
        }
      });
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }

  scrollToBottom() {
    if (this.autoScroll) {
      setTimeout(() => {
        this.logBody.nativeElement.scrollTop =
          this.logBody.nativeElement.scrollHeight + 60;
      }, 0);
    }
  }

  refreshLog() {
    this.logs = [];
  }

  openBlank() {
    window.open(`./logs?ns=${this.ns}&p=${this.podName}`, '_blank');
  }
}
