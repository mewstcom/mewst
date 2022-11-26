# typed: strict
# frozen_string_literal: true

class ApplicationService
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  class BaseError
    extend T::Sig

    sig { returns(String) }
    attr_reader :message

    sig { params(message: String).void }
    def initialize(message:)
      @message = message
    end
  end

  class BaseResult
    extend T::Sig

    sig { returns(T::Array[BaseError]) }
    attr_reader :errors

    sig { params(errors: T::Array[BaseError]).void }
    def initialize(errors: [])
      @errors = errors
    end

    sig { returns(T::Boolean) }
    def ok?
      errors.empty?
    end
  end

  sig { params(errors: ActiveModel::Errors).returns(BaseResult) }
  private def validation_error_result(errors:)
    BaseResult.new(errors: errors.map { |error| BaseError.new(message: error.full_message) })
  end
end
