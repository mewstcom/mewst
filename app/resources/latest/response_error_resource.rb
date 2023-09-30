# typed: strict
# frozen_string_literal: true

class Latest::ResponseErrorResource < Latest::ApplicationResource
  sig { returns(String) }
  attr_reader :message

  sig { params(message: String).void }
  def initialize(message:)
    @message = message
  end
end
