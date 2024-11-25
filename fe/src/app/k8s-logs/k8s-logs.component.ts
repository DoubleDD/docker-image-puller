import {
  CdkVirtualScrollViewport,
  FixedSizeVirtualScrollStrategy,
  ScrollingModule,
  VIRTUAL_SCROLL_STRATEGY,
} from '@angular/cdk/scrolling';
import { CommonModule } from '@angular/common';
import {
  AfterViewInit,
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
import { Observable, Subscription, switchMap } from 'rxjs';
import { K8sService } from '../services/k8s.service';
import { DropdownSelectorComponent } from '../dropdown-selector/dropdown-selector.component';
import { DownloadService } from '../services/download.service';

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
  imports: [
    FormsModule,
    CommonModule,
    ScrollingModule,
    DropdownSelectorComponent,
  ],
  templateUrl: './k8s-logs.component.html',
  styleUrl: './k8s-logs.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    { provide: VIRTUAL_SCROLL_STRATEGY, useClass: CustomVirtualScrollStrategy },
  ],
})
export class K8sLogsComponent implements AfterViewInit, OnInit, OnDestroy {
  @ViewChild(CdkVirtualScrollViewport) viewport!: CdkVirtualScrollViewport;

  selectedContainer: string = '';
  containers: string[] = ['a', 'b', 'c', 'd'];

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
    private downloadService: DownloadService,
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

      // 获取container
      this.logsService
        .getContainers(this.ns, this.podName)
        .subscribe((resp: any) => {
          this.containers = resp.list;
          if (this.containers.length > 0) {
            this.selectedContainer = this.containers[0];
          }
          this.cdr.detectChanges(); //触发一下页面渲染
          this.getLogs();
        });
    });
  }

  ngAfterViewInit(): void {}

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

  /**
   * 下载日志
   */
  download() {
    const blob = new Blob(
      this.logs.map((v) => v + '\n'),
      { type: 'text/plain' },
    );
    this.downloadService.downloadFile(blob, this.podName + '.log');
  }

  getLogs() {
    this.subscription = this.logsService
      .getLogs(this.ns, this.podName, this.selectedContainer)
      .subscribe((log) => this.logHandler(log));
  }

  private logHandler(log: string) {
    if (log.endsWith('\n')) {
      log = log.slice(0, -1);
    }

    this.logs = [...this.logs, ...log.split('\n')];
    this.cdr.detectChanges(); // 手动设置监测更新
    this.viewport.checkViewportSize();
    this.scrollToBottom();
  }

  trackByFn(index: number, _: string): number {
    return index; // 返回列表项的索引
  }

  handleDropdownChange(option: string): void {
    console.log('Selected option:', option);
    // 在这里执行业务逻辑
    this.logs = [];
    this.selectedContainer = option;
    this.cdr.detectChanges();
    this.getLogs();
  }
}
