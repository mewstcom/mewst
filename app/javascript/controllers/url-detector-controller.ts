import { Controller } from '@hotwired/stimulus';
import { FrameElement } from '@hotwired/turbo';

export default class extends Controller {
  static targets = ['linkFormFrame', 'textarea'];
  static values = {
    newLinkPath: String,
  };

  declare readonly linkFormFrameTarget: FrameElement;
  declare readonly newLinkPathValue: String;
  declare readonly textareaTarget: HTMLTextAreaElement;

  detectUrl() {
    const url = this.extractLastUrl(this.textareaTarget.value);

    if (url) {
      this.linkFormFrameTarget.src = `${this.newLinkPathValue}?url=${encodeURIComponent(url)}`;
    } else {
      this.linkFormFrameTarget.src = null;
      this.linkFormFrameTarget.innerHTML = '';
    }
  }

  extractLastUrl(text: string) {
    const urlPattern = /https?:\/\/\S+/g;

    // テキストからすべてのURLを検出
    const urls = text.match(urlPattern);

    // URLが検出された場合、最後のURLを返す
    if (urls && urls.length > 0) {
      return urls[urls.length - 1];
    } else {
      return null;
    }
  }
}
