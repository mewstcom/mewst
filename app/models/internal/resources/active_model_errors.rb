# typed: strict
# frozen_string_literal: true

class Internal::Resources::ActiveModelErrors < Internal::Resources::Base
  root_key :error, :errors

  attribute :code do
    Internal::Resources::Base::ErrorCode::InvalidInputData.serialize
  end

  attribute :message, &:full_message
  attribute :field, &:attribute
end
