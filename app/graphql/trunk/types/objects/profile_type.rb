# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::ProfileType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :atname, String, null: false
  field :avatar_url, String, null: false
  field :description, String, null: false
  field :name, String, null: false
end
