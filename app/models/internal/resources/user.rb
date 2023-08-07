# typed: strict
# frozen_string_literal: true

class Internal::Resources::User < Internal::Resources::Base
  root_key :user, :users

  attributes :id, :locale
end
