# typed: strict
# frozen_string_literal: true

class Resources::Internal::ActiveModelErrors < Resources::Internal::Base
  root_key :error, :errors

  attribute :code do
    Resources::Internal::Base::ErrorCode::InvalidInputData.serialize
  end

  attribute :message, &:full_message
  attribute :field, &:attribute
end
