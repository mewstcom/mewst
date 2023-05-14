# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::Connections::Base < Internal::Types::Objects::Base
  # add `nodes` and `pageInfo` fields, as well as `edge_type(...)` and `node_nullable(...)` overrides
  include GraphQL::Types::Relay::ConnectionBehaviors
end
