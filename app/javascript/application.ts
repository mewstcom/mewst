import '@hotwired/turbo';

import { Application } from '@hotwired/stimulus';
import ujs from '@rails/ujs';
import * as bootstrap from 'bootstrap';

import AutosizeController from './controllers/autosize-controller';
import BlankSlateController from './controllers/blank-slate-controller';
import CharacterCounterController from './controllers/character-counter-controller';
import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import FlashToastController from './controllers/flash-toast-controller';
import FlashToastDispatchController from './controllers/flash-toast-dispatch-controller';
import PostFormController from './controllers/post-form-controller';
import PostModalController from './controllers/post-modal-controller';
import TimelineController from './controllers/timeline-controller';

const application = Application.start();
application.debug = false;
window.Stimulus = application;

Stimulus.register('autosize', AutosizeController);
Stimulus.register('blank-slate', BlankSlateController);
Stimulus.register('character-counter', CharacterCounterController);
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('flash-toast', FlashToastController);
Stimulus.register('flash-toast-dispatch', FlashToastDispatchController);
Stimulus.register('post-form', PostFormController);
Stimulus.register('post-modal', PostModalController);
Stimulus.register('timeline', TimelineController);

const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
Array.from(tooltipTriggerList).map((tooltipTriggerEl) => new bootstrap.Tooltip(tooltipTriggerEl));

ujs.start();
