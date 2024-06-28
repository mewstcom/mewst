# typed: strict
# frozen_string_literal: true

class Mewst::UI::Tabs::Item < Mewst::UI::Base
  sig { params(href: String, active: T::Boolean, class_name: String).void }
  def initialize(href:, active: false, class_name: "")
    @href = href
    @active = active
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :href
  private :href

  sig { returns(T::Boolean) }
  attr_reader :active
  private :active

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
