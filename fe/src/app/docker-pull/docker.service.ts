import { HttpClient, HttpEvent, HttpEventType } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { Manifest } from "./docker-pull.component";

@Injectable({
  providedIn: "root",
})
export class DockerService {
  constructor(private http: HttpClient) {}

  getManifest(imageUrl: string): Observable<Manifest> {
    return this.http.get<Manifest>(
      `/dip/api/docker/manifest?image=${imageUrl}`,
    );
  }

  downloadLayer(digest: string): Observable<Blob> {
    return this.http
      .get(`/dip/api/docker/blob/${digest}`, {
        responseType: "blob",
        reportProgress: true,
        observe: "events",
      })
      .pipe(
        map((event: HttpEvent<Blob>) => {
          if (event.type === HttpEventType.DownloadProgress && event.total) {
            const progress = Math.round((event.loaded / event.total) * 100);
            // You'll need to implement a way to update the progress in the component
          }
          if (event.type === HttpEventType.Response) {
            return event.body as Blob;
          }
          return new Blob();
        }),
      );
  }

  uploadLayer(blob: Blob, digest: string): Observable<any> {
    const formData = new FormData();
    formData.append("layer", blob);
    formData.append("digest", digest);

    return this.http
      .post("/dip/api/docker/upload", formData, {
        reportProgress: true,
        observe: "events",
      })
      .pipe(
        map((event: HttpEvent<any>) => {
          if (event.type === HttpEventType.UploadProgress && event.total) {
            const progress = Math.round((event.loaded / event.total) * 100);
            // You'll need to implement a way to update the progress in the component
          }
          if (event.type === HttpEventType.Response) {
            return event.body;
          }
        }),
      );
  }

  mergeImage(manifest: Manifest, layers: string[]): Observable<any> {
    return this.http.post("/dip/api/docker/merge", { manifest, layers });
  }
}
