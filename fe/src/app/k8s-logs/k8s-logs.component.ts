import { CommonModule } from '@angular/common';
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
  startTime: string = '';
  endTime: string = '';
  playing: boolean = true;
  private subscription!: Subscription;

  @Input() autoScroll: boolean = true;
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

      this.getLogs();
    });
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }

  scrollToBottom() {
    console.log('滚动');

    if (this.autoScroll) {
      const logBodyElement = this.logBody.nativeElement;
      console.log(logBodyElement);
      console.log('scrollHeight:', logBodyElement.scrollHeight);
      console.log('scrollTop:', logBodyElement.scrollTop);
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

  play() {
    if (this.playing) {
      // 正在获取日志，暂停它
      this.subscription.unsubscribe();
    } else {
      // 已经暂停，重新开始
      this.getLogs();
    }
    this.playing = !this.playing;
  }

  getLogs() {
    this.subscription = this.logsService
      .getLogs(this.ns, this.podName, '')
      .subscribe((log) => {
        if (log.trim() != '') {
          const arr: string[] = [...this.logs];
          log
            .trimEnd()
            .split('\n')
            .forEach((l) => arr.push(l));

          if (arr.length >= 500) {
            this.logs = arr.slice(-500);
          } else {
            this.logs = [...arr];
          }
          console.log(this.logs.length);
          this.scrollToBottom();
        }
      });
  }
}
