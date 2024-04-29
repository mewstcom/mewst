import '@hotwired/turbo';

import { Application } from '@hotwired/stimulus';

import AutosizeController from './controllers/autosize-controller';
import CharacterCounterController from './controllers/character-counter-controller';
import DropdownController from './controllers/dropdown-controller';
import FlashToastController from './controllers/flash-toast-controller';
import FlashToastDispatchController from './controllers/flash-toast-dispatch-controller';
import LinkCardFormController from './controllers/link-card-form-controller';
import ModalController from './controllers/modal-controller';

const application = Application.start();
application.debug = false;
window.Stimulus = application;

Stimulus.register('autosize', AutosizeController);
Stimulus.register('character-counter', CharacterCounterController);
Stimulus.register('dropdown', DropdownController);
Stimulus.register('flash-toast', FlashToastController);
Stimulus.register('flash-toast-dispatch', FlashToastDispatchController);
Stimulus.register('link-card-form', LinkCardFormController);
Stimulus.register('modal', ModalController);
