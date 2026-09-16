import { Component, Inject } from '@angular/core';

import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';

export interface TrustSiteRequest {
  /** The origin the decision is about — verified when it came from the event. */
  origin: string;
  app: string;
  icon?: string;

  /**
   * False when the origin is only what the caller claimed. The session
   * transport carries no browser-verified origin, so such a request is shown
   * with a warning and never remembered.
   */
  verified: boolean;

  /** Set when the caller claimed an origin other than the verified one. */
  claimed?: string;

  /** What the site is asking for, once connected. */
  action: 'connect' | 'sign message' | 'sign transaction';
}

@Component({
  selector: 'app-trust-site',
  standalone: true,
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatIconModule,
  ],
  templateUrl: './trust-site.component.html',
  styleUrl: './trust-site.component.scss'
})
export class TrustSiteComponent {
  constructor(
    private dialogRef: MatDialogRef<TrustSiteComponent, boolean>,
    @Inject(MAT_DIALOG_DATA) public req: TrustSiteRequest,
  ) {}

  connect() {
    this.dialogRef.close(true);
  }

  cancel() {
    this.dialogRef.close(false);
  }
}
