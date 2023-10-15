# typed: strict
# frozen_string_literal: true

class Latest::UserSerializer < Latest::ApplicationSerializer
  root_key :user, :users

  attributes :id, :locale, :time_zone
end
