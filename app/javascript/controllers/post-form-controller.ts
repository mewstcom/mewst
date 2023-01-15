import { Controller } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';

import TimelineController from './timeline-controller';
import fetcher from '../utils/fetcher';

export default class extends Controller {
  static outlets = ['timeline'];
  static targets = ['content', 'crossPostToTwitterCheckbox', 'submitButton'];

  contentTarget!: HTMLTextAreaElement;
  crossPostToTwitterCheckboxTarget!: HTMLInputElement;
  submitButtonTarget!: HTMLButtonElement;
  timelineOutlet!: TimelineController | null;

  async submit() {
    this.submitButtonTarget.disabled = true;

    try {
      const content = this.contentTarget.value;
      const crossPostToTwitter = this.crossPostToTwitterCheckboxTarget.checked;

      const response = await fetcher.post('/api/internal/posts', {
        content,
        cross_post_to_twitter: crossPostToTwitter
      });

      this.contentTarget.value = '';
      this.contentTarget.focus();

      setTimeout(() => {
        this.timelineOutlet.element.insertAdjacentHTML('afterbegin', response.postCardHtml);
      }, 0);
    } catch (err) {
      console.error(err);
    }

    this.submitButtonTarget.disabled = false;
  }
}
