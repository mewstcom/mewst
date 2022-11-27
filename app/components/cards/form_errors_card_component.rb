# typed: strict
# frozen_string_literal: true

class Cards::FormErrorsCardComponent < ApplicationComponent
  sig { params(errors: ActiveModel::Errors).void }
  def initialize(errors:)
    @errors = errors
  end

  sig { returns(T::Boolean) }
  private def render?
    !@errors.empty?
  end
end
