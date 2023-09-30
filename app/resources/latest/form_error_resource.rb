# typed: strict
# frozen_string_literal: true

class Latest::FormErrorResource < Latest::ApplicationResource
  sig { params(errors: ActiveModel::Errors).returns(T::Array[T.attached_class]) }
  def self.from_errors(errors:)
    errors.map do |error|
      new(error:)
    end
  end

  sig { params(error: ActiveModel::Error).void }
  def initialize(error:)
    @error = error
  end

  sig { overridable.returns(Latest::ResponseErrorCode) }
  def code
    Latest::ResponseErrorCode::InvalidInputData
  end

  sig { overridable.returns(T.nilable(String)) }
  def field
    error.attribute.to_s
  end

  sig { overridable.returns(String) }
  def message
    error.full_message
  end

  sig { returns(ActiveModel::Error) }
  attr_reader :error
  private :error
end
