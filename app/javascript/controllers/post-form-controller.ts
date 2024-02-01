import { Controller } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';

import TimelineController from './timeline-controller';
import { EventDispatcher } from '../utils/event-dispatcher';
import fetcher from '../utils/fetcher';
import BlankSlateController from './blank-slate-controller';

export default class extends Controller<HTMLFormElement> {
  static outlets = ['timeline', 'blank-slate'];
  static targets = ['content', 'submitButton'];
  static values = {
    successMessage: String,
  };

  declare readonly blankSlateOutlet: BlankSlateController | null;
  declare readonly contentTarget: HTMLTextAreaElement;
  declare readonly hasTimelineOutlet: Boolean;
  declare readonly hasBlankSlateOutlet: Boolean;
  declare readonly submitButtonTarget: HTMLButtonElement;
  declare readonly successMessageValue: string;
  declare readonly timelineOutlet: TimelineController | null;

  async submit() {
    this.submitButtonTarget.disabled = true;

    try {
      const content = this.contentTarget.value;

      const response = await fetcher.post('/api/posts', {
        content,
      });

      this.contentTarget.value = '';
      this.contentTarget.focus();

      setTimeout(() => {
        if (this.hasBlankSlateOutlet) {
          this.blankSlateOutlet.hide();
        }

        if (this.hasTimelineOutlet) {
          this.timelineOutlet.element.insertAdjacentHTML('afterbegin', response.postCardHtml);
        }

        new EventDispatcher('flash-toast:show', {
          type: 'notice',
          messageHtml: this.successMessageValue,
        }).dispatch();

        new EventDispatcher('post-modal:hide').dispatch();
      }, 0);
    } catch (err) {
      console.error(err);
    }

    this.submitButtonTarget.disabled = false;
  }
}
