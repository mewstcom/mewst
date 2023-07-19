# typed: strict
# frozen_string_literal: true

class Resources::Internal::Error < Resources::Internal::Base
  root_key :error, :errors

  attributes :code, :message
end
