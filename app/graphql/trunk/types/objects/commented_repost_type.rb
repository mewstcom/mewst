# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::CommentedRepostType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node
  implements Trunk::Types::Interfaces::Postable
  implements Trunk::Types::Interfaces::Repostable

  field :comment, String, null: false
  field :repostable, Trunk::Types::Interfaces::Repostable, null: false
  field :reposts_count, Integer, null: false
end
