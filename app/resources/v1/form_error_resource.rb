# typed: strict
# frozen_string_literal: true

class V1::FormErrorResource < V1::ApplicationResource
  include Resource::ErrorResponsable

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

  sig { override.returns(V1::ResponseErrorCode) }
  def code
    V1::ResponseErrorCode::InvalidInputData
  end

  sig { overridable.returns(T.nilable(String)) }
  def field
    error.attribute.to_s
  end

  sig { override.returns(String) }
  def message
    error.full_message
  end

  sig { returns(ActiveModel::Error) }
  attr_reader :error
  private :error
end
