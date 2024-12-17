import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { async } from 'rxjs';

@Component({
  selector: 'app-http-client',
  imports: [CommonModule, FormsModule],
  templateUrl: './http-client.component.html',
  styleUrl: './http-client.component.scss',
})
export class HttpClientComponent {
  url: string = ''; // 用于存储 URL
  method: string = 'GET'; // 默认请求方法
  headers: { key: string; value: string }[] = []; // 请求头
  params: { key: string; value: string }[] = []; // 请求参数
  body: string = ''; // 请求体
  response: any = null; // 响应结果

  pk = '';
  pv = '';
  hk = '';
  hv = '';

  constructor(private http: HttpClient) {}

  addParam(key: string, value: string) {
    if (key) {
      this.params.push({ key, value });
      this.pk = '';
      this.pv = '';
    }
  }
  removeParam(index: number) {
    this.params.splice(index, 1);
  }

  addHeader(key: string, value: string) {
    if (key) {
      this.headers.push({ key, value });
      this.hk = '';
      this.hv = '';
    }
  }
  removeHeader(index: number) {
    this.headers.splice(index, 1);
  }
  // 将 params 数组转换为 Record<string, string> 对象
  convertParamsToRecord(
    params: { key: string; value: string }[],
  ): Record<string, string> {
    return params.reduce(
      (acc, param) => {
        if (param.key && param.value) {
          acc[param.key] = param.value;
        }
        return acc;
      },
      {} as Record<string, string>,
    );
  }

  // 发送请求
  sendRequest() {
    try {
      // 构建请求头
      const headers = new Headers();
      this.headers.forEach((header) => {
        headers.append(header.key, header.value);
      });

      // 构建请求参数（仅适用于 GET 请求）
      const queryParams = new URLSearchParams(
        this.convertParamsToRecord(this.params),
      ).toString();
      const fullUrl = queryParams ? `${this.url}?${queryParams}` : this.url;

      // 发送请求
      fetch(fullUrl, {
        method: this.method,
        headers: headers,
        body: this.method === 'GET' ? null : this.body, // GET 请求没有请求体
      }).then(async (response) => {
        const firstLine =
          'HTTP/1.1 ' + response.status + ' ' + response.statusText;
        let respHeaders = '';
        response.headers.forEach((k, v) => {
          console.log(v, k);
          respHeaders += v + ': ' + k + '\n';
        });
        // 解析响应
        const data = await response.text();
        this.response = firstLine + '\n' + respHeaders + '\n' + data;
      });
    } catch (error) {
      console.error('Error sending request:', error);
      this.response = { error: 'Failed to fetch data' };
    }
  }
}
