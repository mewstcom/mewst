# typed: strict
# frozen_string_literal: true

class Icons::UnstampedIconComponent < ApplicationComponent
  erb_template <<~ERB
    <svg
      class="u-fa-icon u-svg-icon"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 448 512"
    >
      <!--! Font Awesome Pro 6.4.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license (Commercial License) Copyright 2023 Fonticons, Inc. -->
      <path d="M192 80A48 48 0 1 0 96 80a48 48 0 1 0 96 0zm-.7 146.1c7.1-11.3 19.4-18.1 32.7-18.1s25.7 6.9 32.7 18.1l49.2 78.6c8.7 14 20.9 22.8 32.2 28.1c17.8 8.4 30 26.4 30 47.1c0 28.7-23.3 52-52 52c-11.1 0-21.2-3.4-29.6-9.2c-19.6-13.6-43.8-17.6-62.4-17.6s-42.8 4-62.4 17.6c-8.4 5.8-18.5 9.2-29.6 9.2c-28.7 0-52-23.3-52-52c0-20.8 12.2-38.8 30-47.1c11.2-5.3 23.4-14.1 32.2-28.1l49.2-78.6zm-89.9 53.2c-2.8 4.5-7.1 7.8-11.8 10.1C55.6 305.4 32 339.9 32 380c0 55.2 44.8 100 100 100c21.2 0 40.8-6.6 56.9-17.8c17.4-12 52.8-12 70.1 0C275.2 473.4 294.8 480 316 480c55.2 0 100-44.8 100-100c0-40.1-23.6-74.6-57.6-90.6c-4.8-2.2-9-5.6-11.8-10.1l-49.1-78.6C281.6 175.4 253.9 160 224 160s-57.6 15.4-73.4 40.7l-49.2 78.6zM304 128a48 48 0 1 0 0-96 48 48 0 1 0 0 96zm144 64a48 48 0 1 0 -96 0 48 48 0 1 0 96 0zM48 240a48 48 0 1 0 0-96 48 48 0 1 0 0 96z"/>
    </svg>
  ERB
end
