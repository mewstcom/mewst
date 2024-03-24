import '@hotwired/turbo';

import { Application } from '@hotwired/stimulus';

import AutosizeController from './controllers/autosize-controller';
import CharacterCounterController from './controllers/character-counter-controller';
import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import DropdownController from './controllers/dropdown-controller';
import FlashToastController from './controllers/flash-toast-controller';
import FlashToastDispatchController from './controllers/flash-toast-dispatch-controller';
import ModalController from './controllers/modal-controller';
import TimelineController from './controllers/timeline-controller';

const application = Application.start();
application.debug = false;
window.Stimulus = application;

Stimulus.register('autosize', AutosizeController);
Stimulus.register('character-counter', CharacterCounterController);
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('dropdown', DropdownController);
Stimulus.register('flash-toast', FlashToastController);
Stimulus.register('flash-toast-dispatch', FlashToastDispatchController);
Stimulus.register('modal', ModalController);
Stimulus.register('timeline', TimelineController);
