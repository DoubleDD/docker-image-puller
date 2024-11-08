import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DockerImagesComponent } from './docker-images.component';

describe('DockerImagesComponent', () => {
  let component: DockerImagesComponent;
  let fixture: ComponentFixture<DockerImagesComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DockerImagesComponent]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DockerImagesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
