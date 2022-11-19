# typed: strict
# frozen_string_literal: true

class Cards::FormErrorsCardComponent < ApplicationComponent
  def initialize(errors:)
    @errors = errors
  end

  private

  sig { returns(T::Boolean) }
  def render?
    !@errors.empty?
  end
end
