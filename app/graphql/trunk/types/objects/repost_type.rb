# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::RepostType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node
  implements Trunk::Types::Interfaces::Postable

  field :repostable, Trunk::Types::Interfaces::Repostable, null: false
end
