# typed: strict
# frozen_string_literal: true

class Mewst::UI::Icon < Mewst::UI::Base
  sig { params(name: String, size: String, class_name: String).void }
  def initialize(name:, size: "16px", class_name: "")
    @name = name
    @size = size
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :name
  private :name

  sig { returns(String) }
  attr_reader :size
  private :size

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
