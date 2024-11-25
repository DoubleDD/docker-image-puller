import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class DownloadService {
  downloadFile(blob: Blob, filename: string) {
    downloadFile(blob, filename);
  }

  saveAsFile(blob: Blob, filename: string) {
    saveAs(blob, filename);
  }
}

function downloadFile(blob: Blob, filename: string) {
  // 创建一个对象 URL
  const url = window.URL.createObjectURL(blob);

  // 创建一个下载链接并点击
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();

  // 下载完成后清理
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}

async function saveAs(blob: Blob, filename: string) {
  const type = blob.type;
  const suffix = filename.substring(filename.lastIndexOf('.'));
  const options = {
    suggestedName: filename, // 默认文件名
    types: [{ [type]: [suffix] }],
  };
  try {
    // 请求用户选择保存文件的位置和文件名
    const fileHandle = await (window as any).showSaveFilePicker(options);

    // 创建一个可写流
    const writableStream = await fileHandle.createWritable();

    // 写入内容到文件
    await writableStream.write(blob);

    // 关闭流，完成保存
    await writableStream.close();

    console.log('文件已成功保存');
  } catch (error) {
    console.error('保存文件时出现错误: ', error);
  }
}
