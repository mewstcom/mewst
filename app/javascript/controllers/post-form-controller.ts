import { Controller } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';

import TimelineController from './timeline-controller';
import fetcher from '../utils/fetcher';

export default class extends Controller {
  static outlets = ['timeline'];
  static targets = ['comment', 'submitButton'];

  commentTarget!: HTMLTextAreaElement;
  submitButtonTarget!: HTMLButtonElement;
  timelineOutlet!: TimelineController | null;

  async submit() {
    this.submitButtonTarget.disabled = true;

    try {
      const comment = this.commentTarget.value;

      const response = await fetcher.post('/api/internal/posts', {
        comment,
      });

      this.commentTarget.value = '';
      this.commentTarget.focus();

      setTimeout(() => {
        this.timelineOutlet.element.insertAdjacentHTML('afterbegin', response.postCardHtml);
      }, 0);
    } catch (err) {
      console.error(err);
    }

    this.submitButtonTarget.disabled = false;
  }
}
