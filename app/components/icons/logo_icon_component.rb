# typed: strict
# frozen_string_literal: true

class Icons::LogoIconComponent < ApplicationComponent
  sig { params(size: String, class_name: String).void }
  def initialize(size:, class_name: "")
    @size = size
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :size
  private :size

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
