import { Controller } from '@hotwired/stimulus';

import { EventDispatcher } from '../utils/event-dispatcher';
import fetcher from '../utils/fetcher';

export default class extends Controller {
  static values = {
    eventName: String,
    path: String,
    payload: Object,
  };

  eventNameValue!: string;
  pathValue!: string;
  payloadValue!: object;

  async connect() {
    const data = await fetcher.post(this.pathValue, this.payloadValue);

    new EventDispatcher(this.eventNameValue, data).dispatch();
  }
}
