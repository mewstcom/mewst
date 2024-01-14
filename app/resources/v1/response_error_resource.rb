# typed: strict
# frozen_string_literal: true

class V1::ResponseErrorResource < V1::ApplicationResource
  include Resource::ErrorResponsable

  sig { override.returns(String) }
  attr_reader :message

  sig { override.returns(V1::ResponseErrorCode) }
  attr_reader :code

  sig { params(code: V1::ResponseErrorCode, message: String).void }
  def initialize(code:, message:)
    @code = code
    @message = message
  end
end
