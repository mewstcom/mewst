# typed: strict
# frozen_string_literal: true

class Internal::ResponseErrorResource < Internal::ApplicationResource
  sig { returns(String) }
  attr_reader :message

  sig { params(message: String).void }
  def initialize(message:)
    @message = message
  end
end
