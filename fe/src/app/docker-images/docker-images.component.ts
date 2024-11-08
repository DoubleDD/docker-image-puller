import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

export interface Image{
  name:string;
  tag?:string;
  description:string;
}
@Component({
  selector: 'app-docker-images',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './docker-images.component.html',
  styleUrl: './docker-images.component.scss'
})
export class DockerImagesComponent {
  isProcessing = false;
  error = '';
  keyword = '';
  images:Image[] = [
    {name:'nginx',description:'nginx'},
    {name:'ubuntu',description:'ubuntu'},
    {name:'openjdk',description:'java17'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'ylns/nginx-empty',tag:'1.19.2',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
    {name:'golang',description:'golang'},
  ];


  search(){

  }
}
