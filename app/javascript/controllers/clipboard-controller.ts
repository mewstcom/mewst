import { Controller } from '@hotwired/stimulus';
import {EventDispatcher} from "../utils/event-dispatcher";

export default class extends Controller<HTMLButtonElement> {
  static values = {
    text: String,
    successMessage: String,
  };

  declare readonly textValue: string;
  declare readonly successMessageValue: string;

  copy() {
    if (!this.textValue) {
      return;
    }

    navigator.clipboard.writeText(this.textValue).then(() => {
      new EventDispatcher('flash-toast:show', {
        type: 'notice',
        messageHtml: this.successMessageValue || 'Copied to clipboard',
      }).dispatch();
    });
  }
}
