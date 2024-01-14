# typed: strict
# frozen_string_literal: true

class V1::ResponseErrorSerializer < V1::ApplicationSerializer
  root_key :error, :errors

  attributes :code, :message
end
