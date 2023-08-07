# typed: strict
# frozen_string_literal: true

class Latest::Presenters::RepostFormErrors < Latest::Presenters::Base
  sig { params(errors: ActiveModel::Errors).void }
  def initialize(errors:)
    @errors = errors
  end

  sig { params(args: T.untyped).returns(T::Hash[Symbol, T.untyped]) }
  def as_json(*args)
    {
      errors: Latest::Resources::ResponseError.new(build_response_errors).to_h
    }
  end

  sig { returns(ActiveModel::Errors) }
  attr_reader :errors
  private :errors

  sig { returns(T::Array[Latest::ResponseError]) }
  private def build_response_errors
    errors.map do |error|
      Latest::ResponseError.new(
        code: Latest::ResponseErrorCode::InvalidInputData,
        field: field(error:),
        message: error.full_message
      )
    end
  end

  sig { params(error: ActiveModel::Error).returns(T.nilable(String)) }
  private def field(error:)
    case error.attribute
    when :target_post
      "post_id"
    when :original_follow
      nil
    else
      error.attribute.to_s
    end
  end
end
