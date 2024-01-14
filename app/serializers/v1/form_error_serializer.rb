# typed: strict
# frozen_string_literal: true

class V1::FormErrorSerializer < V1::ApplicationSerializer
  root_key :error, :errors

  attributes :code, :field, :message
end
