import {
  CdkVirtualScrollViewport,
  FixedSizeVirtualScrollStrategy,
  ScrollingModule,
  VIRTUAL_SCROLL_STRATEGY,
} from '@angular/cdk/scrolling';
import { CommonModule } from '@angular/common';
import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  Input,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';
import { K8sService } from '../services/k8s.service';

/**
 * 自定义策略
 */
export class CustomVirtualScrollStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(50, 250, 500);
  }
}

@Component({
  selector: 'app-k8s-logs',
  imports: [FormsModule, CommonModule, ScrollingModule],
  templateUrl: './k8s-logs.component.html',
  styleUrl: './k8s-logs.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    { provide: VIRTUAL_SCROLL_STRATEGY, useClass: CustomVirtualScrollStrategy },
  ],
})
export class K8sLogsComponent implements OnInit, OnDestroy {
  @ViewChild(CdkVirtualScrollViewport) viewport!: CdkVirtualScrollViewport;
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
    private cdr: ChangeDetectorRef,
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
    if (this.autoScroll) {
      // 平滑滚动到底部
      requestAnimationFrame(() => {
        const ele = this.viewport.getElementRef().nativeElement;
        ele.scrollTo({ top: ele.scrollHeight, behavior: 'smooth' });
      });
    }
  }

  refreshLog() {
    this.logs = [];
    this.cdr.detectChanges(); // 手动设置监测更新
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
        console.log(log);
        if (log.endsWith('\n')) {
          log = log.slice(0, -1);
        }

        this.logs = [...this.logs, ...log.split('\n')];
        this.cdr.detectChanges(); // 手动设置监测更新
        this.viewport.checkViewportSize();
        this.scrollToBottom();
      });
  }

  trackByFn(index: number, _: string): number {
    return index; // 返回列表项的索引
  }
}
