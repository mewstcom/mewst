# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::ProfileType < Internal::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :atname, String, null: false
  field :avatar_url, String, null: false
  field :name, String, null: false
end
