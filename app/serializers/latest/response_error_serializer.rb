# typed: strict
# frozen_string_literal: true

class Latest::ResponseErrorSerializer < Latest::ApplicationSerializer
  root_key :error, :errors

  attributes :code, :field, :message
end
