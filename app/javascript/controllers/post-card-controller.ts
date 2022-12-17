import { Controller } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';

import StickyPostController from './sticky-post-controller';

export default class extends Controller {
  static outlets = ['sticky-post'];
  static values = { stickyPostFrameId: String, postPath: String };

  stickyPostFrameIdValue!: string;
  postPathValue!: string;

  stickyPostOutlet!: StickyPostController | null;

  showAsStickyPost() {
    setTimeout(() => {
      this.stickyPostOutlet.frameTarget.id = this.stickyPostFrameIdValue;
      this.stickyPostOutlet.frameTarget.src = this.postPathValue;
    }, 0);
  }
}
