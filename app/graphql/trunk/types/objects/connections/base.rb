# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::Connections::Base < Trunk::Types::Objects::Base
  # add `nodes` and `pageInfo` fields, as well as `edge_type(...)` and `node_nullable(...)` overrides
  include GraphQL::Types::Relay::ConnectionBehaviors
end
