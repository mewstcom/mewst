# typed: strict
# frozen_string_literal: true

class BlankSlateComponent < ApplicationComponent
  sig { params(message: String, class_name: String).void }
  def initialize(message:, class_name: "")
    @message = message
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :message
  private :message

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
