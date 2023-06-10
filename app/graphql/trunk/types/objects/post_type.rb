# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::PostType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :database_id, String, null: false
  field :postable, Trunk::Types::Interfaces::Postable, null: false
  field :published_at, GraphQL::Types::ISO8601DateTime, null: false

  def database_id
    object.id
  end
end
