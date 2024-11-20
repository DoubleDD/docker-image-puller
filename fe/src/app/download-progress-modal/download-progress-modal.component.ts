import { CommonModule } from '@angular/common';
import { Component, Input, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
    imports: [CommonModule, FormsModule],
    selector: 'app-download-progress-modal',
    templateUrl: './download-progress-modal.component.html',
    styleUrls: ['./download-progress-modal.component.scss']
})
export class DownloadProgressModalComponent implements OnInit {
  @Input() layers: any[] = [];
  @Input() totalSize: number = 0;
  @Input() visible: boolean = false;
  @Input() imageName: string = ''; // 添加 imageName 输入属性

  allLayersCompleted: boolean = false; // 添加 allLayersCompleted 属性
  showButtonAnimation: boolean = false; // 添加 showButtonAnimation 属性

  constructor() {}

  ngOnInit(): void {}

  show() {
    this.visible = true;
  }

  hide() {
    this.visible = false;
  }

  getLayerWidth(layer: any) {
    return (layer.size / this.totalSize) * 100;
  }

  updateLayerProgress(layerIndex: number, progress: number) {
    this.layers[layerIndex].progress = progress;
  }

  updateLayerUploadProgress(layerIndex: number, upProgress: number) {
    this.layers[layerIndex].uProgress = upProgress;
    this.checkAllLayersCompleted();
  }

  checkAllLayersCompleted() {
    const allDone = this.layers.every((layer) => layer.uProgress >= 100);

    if (allDone) {
      setTimeout(() => {
        this.allLayersCompleted = true; // 显示按钮动画
        this.showButtonAnimation = true; // 显示按钮动画
      }, 500);
    }
  }
}
