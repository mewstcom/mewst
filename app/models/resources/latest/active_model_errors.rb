# typed: strict
# frozen_string_literal: true

class Resources::Latest::ActiveModelErrors < Resources::Latest::Base
  root_key :error, :errors

  attribute :code do
    Resources::Latest::Base::ErrorCode::InvalidInputData.serialize
  end

  attribute :message, &:full_message
  attribute :field, &:attribute
end
