# typed: strict
# frozen_string_literal: true

class Latest::RepostFormErrors
  extend T::Sig

  sig { params(errors: ActiveModel::Errors).void }
  def initialize(errors:)
    @errors = errors
  end

  sig { returns(T::Array[Latest::ResponseError]) }
  def build_response_errors
    errors.map do |error|
      Latest::ResponseError.new(
        code: Latest::ResponseErrorCode::InvalidInputData,
        field: field(error:),
        message: error.full_message
      )
    end
  end

  sig { returns(ActiveModel::Errors) }
  attr_reader :errors
  private :errors

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
