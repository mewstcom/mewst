import { Controller } from "@hotwired/stimulus";

import { EventDispatcher } from "../utils/event-dispatcher";

export default class extends Controller {
  static values = {
    messageHtml: String,
    type: String,
  };

  declare readonly messageHtmlValue: string;
  declare readonly typeValue: string;

  connect() {
    new EventDispatcher("flash-toast:show", {
      type: this.typeValue,
      messageHtml: this.messageHtmlValue,
    }).dispatch();
  }
}
