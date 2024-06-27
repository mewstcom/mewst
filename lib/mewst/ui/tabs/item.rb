# typed: strict
# frozen_string_literal: true

class Mewst::UI::Tabs::Item < Mewst::UI::Base
  sig { params(active: T::Boolean, class_name: String).void }
  def initialize(active: false, class_name: "")
    @active = active
    @class_name = class_name
  end

  sig { returns(T::Boolean) }
  attr_reader :active
  private :active

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
