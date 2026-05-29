import "@hotwired/turbo";

import { Application } from "@hotwired/stimulus";

import AutosizeController from "./controllers/autosize-controller";
import CharacterCounterController from "./controllers/character-counter-controller";
import ClipboardController from "./controllers/clipboard-controller";
import DropdownController from "./controllers/dropdown-controller";
import FlashToastController from "./controllers/flash-toast-controller";
import FlashToastDispatchController from "./controllers/flash-toast-dispatch-controller";
import LinkCardFormController from "./controllers/link-card-form-controller";
import ModalController from "./controllers/modal-controller";

declare global {
  interface Window {
    Stimulus: Application;
  }
}

const application = Application.start();
application.debug = false;
window.Stimulus = application;

application.register("autosize", AutosizeController);
application.register("character-counter", CharacterCounterController);
application.register("clipboard", ClipboardController);
application.register("dropdown", DropdownController);
application.register("flash-toast", FlashToastController);
application.register("flash-toast-dispatch", FlashToastDispatchController);
application.register("link-card-form", LinkCardFormController);
application.register("modal", ModalController);
