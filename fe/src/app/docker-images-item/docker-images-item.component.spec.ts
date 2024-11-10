import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DockerImagesItemComponent } from './docker-images-item.component';

describe('DockerImagesItemComponent', () => {
  let component: DockerImagesItemComponent;
  let fixture: ComponentFixture<DockerImagesItemComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DockerImagesItemComponent]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DockerImagesItemComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
