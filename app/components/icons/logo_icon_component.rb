# typed: strict
# frozen_string_literal: true

class Icons::LogoIconComponent < ApplicationComponent
  erb_template <<~ERB
    <%= inline_svg_tag("logo.svg", height: "36px", width: "36px") %>
  ERB
end
