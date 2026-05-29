# typed: strict
# frozen_string_literal: true

class Mewst::UI::Base < ViewComponent::Base
  extend T::Sig

  delegate :inline_svg_tag, to: :helpers
end
