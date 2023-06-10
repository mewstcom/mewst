# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::CommentedPostType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node
  implements Trunk::Types::Interfaces::Postable

  field :comment, String, null: false
  field :reposts_count, Integer, null: false
end
