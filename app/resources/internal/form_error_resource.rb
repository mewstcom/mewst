# typed: strict
# frozen_string_literal: true

class Internal::FormErrorResource < Internal::ApplicationResource
  sig { params(errors: ActiveModel::Errors).returns(T::Array[T.attached_class]) }
  def self.build_from_errors(errors:)
    errors.map do |error|
      new(error:)
    end
  end

  sig { params(error: ActiveModel::Error).void }
  def initialize(error:)
    @error = error
  end

  sig { overridable.returns(String) }
  def message
    error.full_message
  end

  sig { returns(ActiveModel::Error) }
  attr_reader :error
  private :error
end
