# typed: strict
# frozen_string_literal: true

class Internal::Resources::Error < Internal::Resources::Base
  root_key :error, :errors

  attributes :code, :message
end
