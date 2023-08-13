# typed: strict
# frozen_string_literal: true

class Internal::UserSerializer < Internal::ApplicationSerializer
  root_key :user, :users

  attributes :id, :locale
end
