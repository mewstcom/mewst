# typed: strict
# frozen_string_literal: true

class Resources::Internal::User < Resources::Base
  root_key :user, :users

  attributes :id, :locale
end
