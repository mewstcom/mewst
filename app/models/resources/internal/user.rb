# typed: strict
# frozen_string_literal: true

class Resources::Internal::User < Resources::Internal::Base
  root_key :user, :users

  attributes :id, :locale
end
