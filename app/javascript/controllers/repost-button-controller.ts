import { Controller } from '@hotwired/stimulus';

import fetcher from '../utils/fetcher';

export default class extends Controller {
  static values = {
    postId: String,
  };

  postIdValue!: string;

  isReposting: boolean;

  disable() {
    this.element.classList.add('disabled');
  }

  enable() {
    this.element.classList.remove('disabled');
  }

  async repost(_event: any) {
    console.log('this.postIdValue: ', this.postIdValue);

    this.disable();

    try {
      await fetcher.post('/api/internal/repost', {
        post_id: this.postIdValue,
      });

      this.dispatch("reload", { detail: { postId: this.postIdValue } });
    } catch (err) {
      console.error(err);

      // For debugging
      this.dispatch("reload", { detail: { postId: this.postIdValue } });
    }
  }
}
