# typed: strict
# frozen_string_literal: true

class Internal::ResponseErrorSerializer < Internal::ApplicationSerializer
  root_key :error, :errors

  attributes :message
end
