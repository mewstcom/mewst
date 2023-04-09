import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';
import * as bootstrap from 'bootstrap'

import AutosizeController from './controllers/autosize-controller';
import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import EntryFormController from "./controllers/entry-form-controller";
import FollowButtonController from './controllers/follow-button-controller';
import TimelineController from "./controllers/timeline-controller";

window.Stimulus = Application.start();
Stimulus.register('autosize', AutosizeController);
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('entry-form', EntryFormController);
Stimulus.register('follow-button', FollowButtonController);
Stimulus.register('timeline', TimelineController);

const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
Array.from(tooltipTriggerList).map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));

ujs.start();
Turbo.start();
