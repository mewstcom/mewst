# typed: strict
# frozen_string_literal: true

class Mewst::UI::Icon < Mewst::UI::Base
  sig { params(name: String, class_name: String).void }
  def initialize(name:, class_name: "")
    @name = name
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :name
  private :name

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
