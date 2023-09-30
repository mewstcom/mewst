# typed: strict
# frozen_string_literal: true

class Latest::ResponseErrorResource < Latest::ApplicationResource
  include Resource::ErrorResponsable

  sig { override.returns(String) }
  attr_reader :message

  sig { override.returns(Latest::ResponseErrorCode) }
  attr_reader :code

  sig { params(code: Latest::ResponseErrorCode, message: String).void }
  def initialize(code:, message:)
    @code = code
    @message = message
  end
end
