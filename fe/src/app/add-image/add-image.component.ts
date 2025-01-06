import { CommonModule } from '@angular/common';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Component } from '@angular/core';
import {
  FormBuilder,
  FormGroup,
  FormsModule,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { Router } from '@angular/router';

@Component({
  selector: 'app-add-image',
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule, // 确保 ReactiveFormsModule 在这里
  ],
  templateUrl: './add-image.component.html',
  styleUrl: './add-image.component.scss',
})
export class AddImageComponent {
  form: FormGroup; // 定义表单组

  constructor(
    private fb: FormBuilder,
    private http: HttpClient,
    private router: Router,
  ) {
    // 初始化表单
    this.form = this.fb.group({
      name: ['', Validators.required], // 名称字段，必填
      oldImage: ['', [Validators.required]], // 邮箱字段，必填且需符合邮箱格式
      newImage: ['', Validators.required], // 消息字段，必填
    });
  }

  // 提交表单
  onSubmit() {
    if (this.form.valid) {
      const formData = `name=${this.form.get('name')?.value}&oldImage=${this.form.get('oldImage')?.value}&newImage=${this.form.get('newImage')?.value}`;
      console.log('Form Data:', formData);
      //发送 HTTP POST 请求
      // 设置请求头
      const headers = new HttpHeaders().set(
        'Content-Type',
        'application/x-www-form-urlencoded',
      );
      this.http
        .post('/dip/api/docker/images', formData, { headers })
        .subscribe({
          next: (_) => {
            // 跳转到 /docker 路由
            this.router.navigate(['/docker']);
          },
          error: (error) => {
            console.error('Error submitting form:', error);
            alert('Error submitting form. Please try again.');
          },
        });
    } else {
      alert('Please fill out the form correctly.');
    }
  }
}
