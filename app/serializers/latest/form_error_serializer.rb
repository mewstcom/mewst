# typed: strict
# frozen_string_literal: true

class Latest::FormErrorSerializer < Latest::ApplicationSerializer
  root_key :error, :errors

  attributes :code, :field, :message
end
