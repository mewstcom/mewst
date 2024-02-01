# typed: strict
# frozen_string_literal: true

class Icons::StampedIconComponent < ApplicationComponent
  erb_template <<~ERB
    <svg
      class="u-fa-icon u-svg-icon"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 448 512"
    >
      <!--! Font Awesome Pro 6.4.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license (Commercial License) Copyright 2023 Fonticons, Inc. -->
      <path d="M96 80a48 48 0 1 1 96 0A48 48 0 1 1 96 80zm1.7 206c6.2-2.3 11.8-6.3 15-12.2l40-71.9c14.4-25.9 41.7-42 71.3-42s56.9 16.1 71.3 42l40 71.9c3.2 5.8 8.8 9.9 15 12.2c38.3 14 65.7 50.8 65.7 94c0 55.2-44.8 100-100 100c-21.2 0-40.8-6.6-56.9-17.8c-17.4-12-52.8-12-70.1 0C172.8 473.4 153.2 480 132 480C76.8 480 32 435.2 32 380c0-43.2 27.4-80 65.7-94zM304 32a48 48 0 1 1 0 96 48 48 0 1 1 0-96zm48 160a48 48 0 1 1 96 0 48 48 0 1 1 -96 0zM48 144a48 48 0 1 1 0 96 48 48 0 1 1 0-96z"/>
    </svg>
  ERB
end
