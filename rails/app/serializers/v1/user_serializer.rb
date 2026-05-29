# typed: strict
# frozen_string_literal: true

class V1::UserSerializer < V1::ApplicationSerializer
  root_key :user, :users

  attributes :id, :locale, :time_zone
end
