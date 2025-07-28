export class EventDispatcher<T = unknown> {
  event!: CustomEvent<T>;

  constructor(eventName: string, detail: T) {
    this.event = new CustomEvent(eventName, {
      detail,
    });
  }

  dispatch() {
    document.dispatchEvent(this.event);
  }
}
